import base64
import hashlib
import hmac
import json
from datetime import datetime
from pathlib import Path

from fastapi.testclient import TestClient

from lqvoice import prompt, twilio
from lqvoice.bot import for_voice
from lqvoice.config import Config
from lqvoice.server import create_app
from lqvoice.tickets import Ticket, Tickets


def config(**overrides) -> Config:
    values = dict(
        stt="nos-whisper",
        stt_language="gl",
        whisper_model="",
        tts="nos",
        tts_voice="brais",
        llm="anthropic",
        llm_model="claude-haiku-5-5",
        llm_base_url="",
        llm_api_key="key",
        elevenlabs_api_key="",
        deepgram_api_key="",
        device="cpu",
        models_dir=Path("/tmp"),
        mcp_url="http://api:8000/mcp",
        voice_api_key="voice-key",
        public_url="https://reservas.example.gal",
        max_call_seconds=600,
        max_calls=2,
        turn_silence=0.6,
        twilio_account_sid="",
        twilio_auth_token="",
        log_level="WARNING",
    )
    return Config(**{**values, **overrides})


def test_health_config_and_session():
    with TestClient(create_app(config(), warm_up=False)) as client:
        assert client.get("/health").json()["status"] == "ok"
        runtime = client.get("/config.json").json()
        assert runtime["enabled"] is True
        assert runtime["thirdParties"] == {"voice": [], "text": ["Anthropic"]}
        origin = {"Origin": "https://reservas.example.gal"}
        assert client.post("/session", headers={"Origin": "https://evil.example"}).status_code == 403
        assert client.post("/session", headers=origin).json()["ticket"]
        assert client.post("/session", headers=origin).json()["ticket"]
        # VOICE_MAX_CALLS=2: two pending tickets fill the line
        assert client.post("/session", headers=origin).json() == {"error": "busy"}


def test_twilio_routes_only_when_configured():
    with TestClient(create_app(config(), warm_up=False)) as client:
        assert client.post("/twilio").status_code in (404, 405)
    with TestClient(create_app(config(twilio_account_sid="AC1", twilio_auth_token="tok"), warm_up=False)) as client:
        assert client.post("/twilio", data={"From": "+34600000000"}).status_code == 403


def test_tickets_are_single_use():
    tickets = Tickets()
    key = tickets.issue(Ticket(kind="web", headers={"Authorization": "Bearer x"}, signed_in=True))
    assert tickets.pending() == 1
    assert tickets.redeem(key).signed_in
    assert tickets.redeem(key) is None
    assert tickets.redeem(None) is None
    assert Tickets(ttl=-1).redeem(Tickets(ttl=-1).issue(Ticket(kind="web", headers={}))) is None


def test_twilio_signature():
    params = {"From": "+34600000000", "CallSid": "CA1"}
    url = "https://reservas.example.gal/voice/twilio"
    payload = url + "CallSidCA1From+34600000000"
    signature = base64.b64encode(hmac.new(b"tok", payload.encode(), hashlib.sha1).digest()).decode()
    assert twilio.valid_signature("tok", url, params, signature)
    assert not twilio.valid_signature("tok", url, {**params, "From": "+34611111111"}, signature)
    assert 'name="ticket" value="abc"' in twilio.stream_twiml("wss://x/voice/twilio/ws", "abc")


def test_tools_by_call():
    assert "book_turn" not in prompt.tools_for(prompt.Call(kind="web"))
    assert "book_turn" in prompt.tools_for(prompt.Call(kind="web", signed_in=True))
    assert "phone_verify_code" in prompt.tools_for(prompt.Call(kind="phone"))
    assert "sign_in" not in prompt.tools_for(prompt.Call(kind="phone"))


def test_prompt_has_today_and_tomorrow():
    now = datetime(2026, 11, 26, 18, 5, tzinfo=prompt.TIMEZONE)
    system = prompt.system_prompt(prompt.Call(kind="web"), [{"name": "Concello de Lalín", "slug": "lalin"}], now)
    assert "xoves, 26 de novembro de 2026 (2026-11-26)" in system
    assert "Mañá é venres, 27 de novembro" in system
    assert "Concello de Lalín (lalin)" in system
    assert "non iniciou sesión" in system


def test_tool_output_for_voice():
    output = json.dumps(
        {
            "facilities": [
                {
                    "name": "Pista 1",
                    "logo_url": "https://x/storage/...",
                    "free_turns": [{"starts_at": "2026-11-24T16:30:00+01:00", "ends_at": "2026-11-24T17:00:00Z"}],
                }
            ]
        }
    )
    facility = json.loads(for_voice(output))["facilities"][0]
    assert "logo_url" not in facility
    assert facility["free_turns"] == [{"starts_at": "2026-11-24T16:30:00+01:00", "local_time": "16:30-18:00"}]
    assert facility["free_times"] == "16:30-18:00"
    assert for_voice("not json") == "not json"


def test_third_parties():
    from lqvoice.config import third_parties

    local = config(llm="openai-compatible", llm_base_url="http://ollama:11434/v1")
    assert third_parties(local) == {"voice": [], "text": []}
    assert third_parties(config(llm="openai-compatible", llm_base_url="http://192.168.1.5:8000/v1"))["text"] == []
    remote = config(
        llm="openai-compatible", llm_base_url="https://api.together.xyz/v1", stt="elevenlabs", tts="elevenlabs"
    )
    assert third_parties(remote) == {"voice": ["ElevenLabs"], "text": ["api.together.xyz", "ElevenLabs"]}


def test_empty_settings_take_their_defaults(monkeypatch):
    from lqvoice.config import load

    monkeypatch.setenv("VOICE_LLM_API_KEY", "key")
    monkeypatch.setenv("VOICE_LLM_MODEL", "")
    monkeypatch.setenv("VOICE_TTS_VOICE", "")
    monkeypatch.setenv("VOICE_STT", "")
    loaded = load()
    assert (loaded.llm_model, loaded.tts_voice, loaded.stt) == ("claude-haiku-5-5", "brais", "nos-whisper")
