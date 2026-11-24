"""End-to-end smoke test of the voice line, as a browser would call it.

Says a Galician question with a Nós voice, streams it in real time over the WebSocket
(the web panel's protocol: protobuf frames, RTVI messages), and checks that the agent
looked the facilities up and answered with audio that, transcribed back, names one.

    uv run python e2e/smoke.py --url http://localhost:8000 --origin http://localhost

Prints one JSON line with the measurements; exits 1 when a check fails.
"""

import argparse
import asyncio
import json
import re
import sys
import time
import uuid

import aiohttp
import numpy as np
from pipecat.frames.protobufs import frames_pb2

from lqvoice import config as settings
from lqvoice import models

RATE = 16000
CHUNK = RATE // 50  # 20 ms, as the browser sends it


def rtvi(kind: str, data: dict | None = None) -> bytes:
    message = {"label": "rtvi-ai", "type": kind, "id": str(uuid.uuid4())[:8], "data": data or {}}
    frame = frames_pb2.Frame()
    frame.message.data = json.dumps(message)
    return frame.SerializeToString()


def audio(pcm: bytes) -> bytes:
    frame = frames_pb2.Frame()
    frame.audio.audio = pcm
    frame.audio.sample_rate = RATE
    frame.audio.num_channels = 1
    return frame.SerializeToString()


def say(config: settings.Config, text: str) -> bytes:
    checkpoint, voice_config = models.nos_voice(config.models_dir, config.tts_voice if config.tts == "nos" else "brais")
    shared = models.vits(str(checkpoint), str(voice_config), models.device(config.device))
    wav = np.asarray(shared.model.tts(text), dtype=np.float32)
    return (np.clip(wav, -1, 1) * 32767).astype(np.int16).tobytes()


def transcribe(config: settings.Config, pcm: bytes) -> str:
    path = str(models.nos_whisper_ct2(config.models_dir))
    model = models.whisper(path, models.device(config.device)).model
    samples = np.frombuffer(pcm, dtype=np.int16).astype(np.float32) / 32768.0
    segments, _ = model.transcribe(samples, language="gl")
    return " ".join(s.text.strip() for s in segments)


class Call:
    def __init__(self):
        self.events: list[tuple[float, str, dict]] = []
        self.bot_audio: list[tuple[float, bytes]] = []
        self.greeted = asyncio.Event()

    async def receive(self, ws):
        async for msg in ws:
            if msg.type != aiohttp.WSMsgType.BINARY:
                continue
            frame = frames_pb2.Frame.FromString(msg.data)
            kind = frame.WhichOneof("frame")
            now = time.monotonic()
            if kind == "audio":
                self.bot_audio.append((now, frame.audio.audio))
            elif kind == "message":
                message = json.loads(frame.message.data)
                self.events.append((now, message.get("type", ""), message.get("data") or {}))
                if message.get("type") == "bot-stopped-speaking":
                    self.greeted.set()

    def after(self, t: float, kind: str) -> list[dict]:
        return [data for when, k, data in self.events if when >= t and k == kind]

    def answered(self, since: float) -> bool:
        """The agent ran a tool, wrote its answer with the result and finished saying it."""
        kinds = [k for when, k, _ in self.events if when >= since]

        def last(kind: str) -> int:
            return len(kinds) - 1 - kinds[::-1].index(kind) if kind in kinds else -1

        tool = last("llm-function-call-stopped")
        return -1 < tool < last("bot-llm-started") < last("bot-llm-stopped") < last("bot-stopped-speaking")


async def stream(ws, pcm: bytes, seconds_of_silence: float = 0.0):
    pcm += b"\0\0" * int(RATE * seconds_of_silence)
    start = time.monotonic()
    for i, offset in enumerate(range(0, len(pcm), CHUNK * 2)):
        await ws.send_bytes(audio(pcm[offset : offset + CHUNK * 2]))
        await asyncio.sleep(max(0.0, start + (i + 1) * 0.02 - time.monotonic()))


async def silence_until(ws, done: asyncio.Event, timeout: float):
    """Keep the line open with silence, as a quiet microphone does."""
    until = time.monotonic() + timeout
    while not done.is_set() and time.monotonic() < until:
        await stream(ws, b"\0\0" * CHUNK)


async def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://localhost:8000")
    parser.add_argument("--origin", default="http://localhost")
    parser.add_argument("--question", default="Que pistas de pádel hai libres mañá pola tarde en Lalín?")
    parser.add_argument("--expect", default="pádel,padel,lalín,arena,pista", help="any of these in the answer")
    parser.add_argument("--timeout", type=float, default=60)
    args = parser.parse_args()
    config = settings.load()

    question = await asyncio.to_thread(say, config, args.question)
    async with aiohttp.ClientSession(headers={"Origin": args.origin}) as http:
        async with http.get(f"{args.url}/health") as response:
            server = await response.json()
        async with http.post(f"{args.url}/session") as response:
            response.raise_for_status()
            ticket = (await response.json())["ticket"]
        ws_url = args.url.replace("http", "ws", 1) + f"/ws?token={ticket}"
        async with http.ws_connect(ws_url) as ws:
            call = Call()
            receiver = asyncio.create_task(call.receive(ws))
            await ws.send_bytes(rtvi("client-ready", {"version": "2.3.0", "about": {"library": "smoke-test"}}))
            # the greeting
            greeted = time.monotonic()
            await silence_until(ws, call.greeted, args.timeout)
            greeting_seconds = time.monotonic() - greeted
            await asyncio.sleep(0.3)

            asked = time.monotonic()
            await stream(ws, question)
            end_of_speech = time.monotonic()
            # the answer: the agent may speak in several turns (e.g. "looking…", then the result)
            answered = None
            while time.monotonic() - end_of_speech < args.timeout:
                await stream(ws, b"\0\0" * CHUNK)
                spoken = [t for t, _ in call.bot_audio if t > end_of_speech]
                if spoken and answered is None:
                    answered = spoken[0]
                if call.answered(end_of_speech):
                    break
            await ws.close()
            receiver.cancel()

    answer_audio = b"".join(pcm for t, pcm in call.bot_audio if t > end_of_speech)
    heard = await asyncio.to_thread(transcribe, config, answer_audio) if answer_audio else ""
    tools = [d.get("function_name") for d in call.after(asked, "llm-function-call-started")]
    # each segment comes as "new" and again "completed": its latest text, once
    segments = {d.get("segment_id"): d.get("text", "") for d in call.after(asked, "bot-output")}
    said = [text for text in segments.values() if text]
    result = {
        "stt": server.get("stt"),
        "tts": server.get("tts"),
        "llm": server.get("llm"),
        "greeting_seconds": round(greeting_seconds, 2),
        "tools": tools,
        "first_audio_after_speech_s": round(answered - end_of_speech, 3) if answered else None,
        "answer_audio_s": round(len(answer_audio) / 2 / RATE, 1),
        "agent_said": " ".join(said)[:400],
        "heard_back": heard[:400],
    }
    checks = {
        "called search_facilities": "search_facilities" in tools,
        "answered with audio": len(answer_audio) > RATE,
        "names a facility": any(w in heard.lower() for w in args.expect.split(",")),
        "offers a free turn": bool(re.search(r"\b\d{1,2}:\d{2}\b", " ".join(said))),
    }
    result["checks"] = checks
    print(json.dumps(result, ensure_ascii=False))
    return 0 if all(checks.values()) else 1


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
