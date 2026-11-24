"""LibreQuenda's voice line: a browser call over a WebSocket (behind Caddy at /voice/*), or a phone call through Twilio.

Nothing is recorded: audio and transcripts live in memory for the call and go with it.
"""

import asyncio
import sys
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request, WebSocket
from fastapi.responses import JSONResponse, Response
from loguru import logger
from pipecat.runner.utils import parse_telephony_websocket
from pipecat.serializers.protobuf import ProtobufFrameSerializer
from pipecat.serializers.twilio import TwilioFrameSerializer
from pipecat.transports.websocket.fastapi import FastAPIWebsocketParams, FastAPIWebsocketTransport

from lqvoice import config as settings
from lqvoice import prompt, providers, twilio
from lqvoice.bot import run_call
from lqvoice.tickets import Ticket, Tickets


class State:
    def __init__(self, config: settings.Config):
        self.config = config
        self.ready = asyncio.Event()
        self.failed: str | None = None
        self.tickets = Tickets()
        self.calls = 0

    def busy(self) -> bool:
        return self.calls + self.tickets.pending() >= self.config.max_calls


def create_app(config: settings.Config | None = None, warm_up: bool = True) -> FastAPI:
    config = config or settings.load()
    logger.remove()
    logger.add(sys.stderr, level=config.log_level, serialize=True)
    state = State(config)

    @asynccontextmanager
    async def lifespan(_app: FastAPI):
        logger.info("voice line starting", stt=config.stt, tts=config.tts, llm=config.llm, model=config.llm_model)

        async def load():
            try:
                if warm_up:
                    await asyncio.to_thread(providers.warm_up, config)
                state.ready.set()
                logger.info("voice line ready")
            except Exception as e:
                state.failed = str(e)
                logger.exception("could not load the models")

        task = asyncio.create_task(load())
        yield
        task.cancel()

    app = FastAPI(title="LibreQuenda voice", lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)

    @app.get("/health")
    async def health():
        body = {"stt": config.stt, "tts": config.tts, "llm": config.llm, "calls": state.calls}
        if state.failed:
            return JSONResponse({**body, "status": "failed", "error": state.failed}, status_code=503)
        if not state.ready.is_set():
            return JSONResponse({**body, "status": "loading"}, status_code=503)
        return {**body, "status": "ok"}

    @app.get("/config.json")
    async def runtime_config():
        # the web shows the call button only when this answers (the voice profile is on)
        return JSONResponse(
            {
                "enabled": True,
                "language": "gl",
                "maxCallSeconds": config.max_call_seconds,
                "thirdParties": settings.third_parties(config),
                # signed out, a web call can still book with a name and a phone number
                "guestBooking": config.web_guest_booking,
            },
            headers={"Cache-Control": "no-store"},
        )

    def origin_allowed(origin: str | None) -> bool:
        return origin is None or origin.rstrip("/") == config.public_url

    @app.post("/session")
    async def session(request: Request):
        if not origin_allowed(request.headers.get("origin")):
            return JSONResponse({"error": "origin"}, status_code=403)
        if not state.ready.is_set():
            return JSONResponse({"error": "loading"}, status_code=503)
        if state.busy():
            return JSONResponse({"error": "busy"}, status_code=503)
        authorization = request.headers.get("authorization", "")
        signed_in = authorization.lower().startswith("bearer ") and len(authorization) > len("bearer ")
        if signed_in:
            headers = {"Authorization": authorization}
        elif config.web_guest_booking:
            headers = {"X-Voice-Key": config.voice_api_key, "X-Web-Guest": "1"}
        else:
            headers = {}
        ticket = state.tickets.issue(Ticket(kind="web", headers=headers, signed_in=signed_in))
        return JSONResponse({"ticket": ticket}, headers={"Cache-Control": "no-store"})

    @app.websocket("/ws")
    async def web_call(websocket: WebSocket):
        ticket = state.tickets.redeem(websocket.query_params.get("token"))
        if ticket is None or ticket.kind != "web" or not origin_allowed(websocket.headers.get("origin")):
            await websocket.close(code=4403)
            return
        await websocket.accept()
        transport = FastAPIWebsocketTransport(
            websocket=websocket,
            params=FastAPIWebsocketParams(
                audio_in_enabled=True,
                audio_out_enabled=True,
                add_wav_header=False,
                serializer=ProtobufFrameSerializer(),
                allowed_origins=[],
            ),
        )
        await _run(
            state,
            transport,
            prompt.Call(kind="web", signed_in=ticket.signed_in, guest_booking=config.web_guest_booking),
            ticket.headers,
            True,
        )

    if config.twilio_enabled:

        @app.post("/twilio")
        async def twilio_incoming(request: Request):
            form = {k: str(v) for k, v in (await request.form()).items()}
            url = f"{config.public_url}/voice/twilio"
            signature = request.headers.get("x-twilio-signature", "")
            if not twilio.valid_signature(config.twilio_auth_token, url, form, signature):
                return Response(status_code=403)
            if not state.ready.is_set() or state.busy():
                return Response(
                    '<?xml version="1.0" encoding="UTF-8"?><Response><Say language="es-ES">'
                    "A liña está ocupada. Proba de novo nun momento.</Say><Hangup/></Response>",
                    media_type="application/xml",
                )
            caller = form.get("From", "")
            headers = {"X-Voice-Key": config.voice_api_key, "X-Caller-Phone": caller}
            ticket = state.tickets.issue(Ticket(kind="phone", headers=headers))
            stream = config.public_url.replace("https://", "wss://").replace("http://", "ws://") + "/voice/twilio/ws"
            return Response(twilio.stream_twiml(stream, ticket), media_type="application/xml")

        @app.websocket("/twilio/ws")
        async def phone_call(websocket: WebSocket):
            await websocket.accept()
            _, call_data = await parse_telephony_websocket(websocket)
            ticket = state.tickets.redeem((call_data.get("body") or {}).get("ticket"))
            if ticket is None or ticket.kind != "phone":
                await websocket.close(code=4403)
                return
            serializer = TwilioFrameSerializer(
                stream_sid=call_data["stream_id"],
                call_sid=call_data["call_id"],
                account_sid=config.twilio_account_sid,
                auth_token=config.twilio_auth_token,
            )
            transport = FastAPIWebsocketTransport(
                websocket=websocket,
                params=FastAPIWebsocketParams(
                    audio_in_enabled=True,
                    audio_out_enabled=True,
                    add_wav_header=False,
                    serializer=serializer,
                    allowed_origins=[],
                ),
            )
            await _run(state, transport, prompt.Call(kind="phone"), ticket.headers, False)

    return app


async def _run(state: State, transport, call: prompt.Call, headers: dict[str, str], greet_on_ready: bool) -> None:
    state.calls += 1
    logger.info("call started", kind=call.kind, signed_in=call.signed_in, calls=state.calls)
    try:
        await run_call(state.config, transport, call, headers, greet_on_ready=greet_on_ready)
    except Exception:
        logger.exception("call failed")
    finally:
        state.calls -= 1
