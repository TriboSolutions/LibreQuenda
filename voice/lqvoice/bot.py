"""One call: transport -> Silero VAD -> STT -> LLM with LibreQuenda's MCP tools -> TTS -> transport."""

import asyncio
import json
import time
from datetime import datetime

import aiohttp
from loguru import logger
from mcp import ClientSession
from mcp.client.session_group import StreamableHttpParameters
from mcp.client.streamable_http import streamable_http_client
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.frames.frames import EndFrame, TTSSpeakFrame
from pipecat.observers.user_bot_latency_observer import UserBotLatencyObserver
from pipecat.pipeline.pipeline import Pipeline
from pipecat.pipeline.worker import PipelineParams, PipelineWorker
from pipecat.processors.aggregators.llm_context import LLMContext
from pipecat.processors.aggregators.llm_response_universal import LLMContextAggregatorPair, LLMUserAggregatorParams
from pipecat.processors.frameworks.rtvi.observer import RTVIFunctionCallReportLevel, RTVIObserverParams
from pipecat.services.mcp_service import MCPClient
from pipecat.transports.base_transport import BaseTransport
from pipecat.turns.user_stop.speech_timeout_user_turn_stop_strategy import SpeechTimeoutUserTurnStopStrategy
from pipecat.turns.user_turn_strategies import UserTurnStrategies
from pipecat.workers.runner import WorkerRunner

from lqvoice import prompt, providers
from lqvoice.config import Config

COUNCILS_TTL = 600
_councils: tuple[float, list[dict]] = (0.0, [])


async def councils(mcp_url: str) -> list[dict]:
    """The installation's councils, for the prompt and the STT's vocabulary; cached for ten minutes."""
    global _councils
    fetched, cached = _councils
    if cached and time.monotonic() - fetched < COUNCILS_TTL:
        return cached
    try:
        async with streamable_http_client(mcp_url) as (read, write, *_), ClientSession(read, write) as session:
            await session.initialize()
            result = await session.call_tool("list_councils", {})
        found = [{"name": c["name"], "slug": c["slug"]} for c in json.loads(result.content[0].text)["councils"]]
    except Exception as e:
        logger.warning(f"could not list the councils: {e}")
        return cached
    _councils = (time.monotonic(), found)
    return found


def for_voice(output: str) -> str:
    """Tool output for a spoken answer: without presigned image links (long, costly, useless aloud), and with
    each turn's local time spelled out ("15:00-16:30"): models misread "16:30:00+01:00" and mix up turns."""
    try:
        data = json.loads(output)
    except ValueError:
        return output

    def clean(value):
        if isinstance(value, list):
            return [clean(v) for v in value]
        if not isinstance(value, dict):
            return value
        out = {k: clean(v) for k, v in value.items() if not k.endswith("_url")}
        try:
            start = datetime.fromisoformat(out["starts_at"]).astimezone(prompt.TIMEZONE)
            end = datetime.fromisoformat(out.pop("ends_at")).astimezone(prompt.TIMEZONE)
        except (KeyError, TypeError, ValueError):
            return out
        out["local_time"] = f"{start:%H:%M}-{end:%H:%M}"
        return out

    return json.dumps(clean(data), ensure_ascii=False, separators=(",", ":"))


async def run_call(
    config: Config,
    transport: BaseTransport,
    call: prompt.Call,
    mcp_headers: dict[str, str],
    *,
    greet_on_ready: bool,
    audio_out_sample_rate: int = 16000,
) -> None:
    started = time.monotonic()
    found = await councils(config.mcp_url)
    tools_allowed = prompt.tools_for(call)
    async with aiohttp.ClientSession() as http:
        stt = providers.make_stt(config, http, prompt.hotwords(found) if found else None)
        tts = providers.make_tts(config)
        llm = providers.make_llm(config, prompt.system_prompt(call, found))
        mcp = MCPClient(
            server_params=StreamableHttpParameters(url=config.mcp_url, headers=mcp_headers),
            tools_filter=tools_allowed,
            tools_output_filters={name: for_voice for name in tools_allowed},
        )
        context = LLMContext(tools=await mcp.tools())
        user, assistant = LLMContextAggregatorPair(
            context,
            user_params=LLMUserAggregatorParams(
                vad_analyzer=SileroVADAnalyzer(),
                user_turn_strategies=UserTurnStrategies(
                    stop=[SpeechTimeoutUserTurnStopStrategy(user_speech_timeout=config.turn_silence)]
                ),
            ),
        )
        pipeline = Pipeline([transport.input(), stt, user, llm, tts, transport.output(), assistant])

        latency = UserBotLatencyObserver()

        @latency.event_handler("on_latency_measured")
        async def on_latency(_observer, seconds: float):
            logger.info(f"turn latency {seconds:.3f}s", latency=round(seconds, 3))

        worker = PipelineWorker(
            pipeline,
            params=PipelineParams(audio_in_sample_rate=16000, audio_out_sample_rate=audio_out_sample_rate),
            observers=[latency],
            # the caption panel says "searching…" while a tool runs: only its name, never its data
            rtvi_observer_params=RTVIObserverParams(function_call_report_level={"*": RTVIFunctionCallReportLevel.NAME}),
            idle_timeout_secs=120,
        )
        runner = WorkerRunner(handle_sigint=False)
        await runner.add_workers(worker)

        async def greet():
            await worker.queue_frames([TTSSpeakFrame(prompt.GREETING)])

        if greet_on_ready:

            @worker.rtvi.event_handler("on_client_ready")
            async def on_client_ready(_rtvi):
                await greet()

        else:

            @transport.event_handler("on_client_connected")
            async def on_client_connected(_transport, _client):
                await greet()

        @transport.event_handler("on_client_disconnected")
        async def on_client_disconnected(_transport, _client):
            await runner.cancel()

        async def time_limit():
            await asyncio.sleep(config.max_call_seconds)
            logger.info("call reached its time limit")
            await worker.queue_frames([TTSSpeakFrame(prompt.TIME_UP, append_to_context=False), EndFrame()])

        limit = asyncio.create_task(time_limit())
        try:
            await runner.run()
        finally:
            limit.cancel()
            logger.info(
                "call ended", kind=call.kind, signed_in=call.signed_in, seconds=round(time.monotonic() - started)
            )
