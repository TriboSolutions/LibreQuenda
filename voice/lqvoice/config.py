"""The voice line's settings, all from the environment (docs/voice.md lists them)."""

import ipaddress
import os
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import urlparse

STT_PROVIDERS = ("nos-whisper", "nos-wav2vec2", "whisper", "elevenlabs", "deepgram")
TTS_PROVIDERS = ("nos", "piper", "elevenlabs")
LLM_PROVIDERS = ("anthropic", "openai-compatible", "openai")

DEFAULT_LLM_MODEL = {"anthropic": "claude-haiku-5-5", "openai": "gpt-5-mini", "openai-compatible": "qwen3:8b"}
DEFAULT_TTS_VOICE = {"nos": "brais", "piper": "es_ES-davefx-medium", "elevenlabs": ""}


def _env(name: str, default: str = "") -> str:
    """NAME, or the default when it is unset or empty (compose passes ${NAME:-} as "")."""
    return os.environ.get(name, "").strip() or default


def _secret(name: str) -> str:
    """NAME, or the contents of the file NAME_FILE points to (Docker secrets, key files)."""
    path = _env(f"{name}_FILE")
    if path:
        return Path(path).read_text().strip()
    return _env(name)


def _choice(name: str, default: str, allowed: tuple[str, ...]) -> str:
    value = _env(name, default).lower()
    if value not in allowed:
        raise ValueError(f"{name} must be one of {', '.join(allowed)}, got '{value}'")
    return value


@dataclass(frozen=True)
class Config:
    stt: str
    stt_language: str  # "gl", or "auto" to detect it (multilingual models only)
    whisper_model: str
    tts: str
    tts_voice: str
    llm: str
    llm_model: str
    llm_base_url: str
    llm_api_key: str = field(repr=False)
    elevenlabs_api_key: str = field(repr=False)
    deepgram_api_key: str = field(repr=False)
    device: str  # auto, cuda or cpu
    models_dir: Path
    mcp_url: str
    voice_api_key: str = field(repr=False)  # sent as X-Voice-Key on phone calls
    public_url: str
    max_call_seconds: int
    max_calls: int
    turn_silence: float
    twilio_account_sid: str
    twilio_auth_token: str = field(repr=False)
    log_level: str

    @property
    def twilio_enabled(self) -> bool:
        return bool(self.twilio_account_sid and self.twilio_auth_token and self.voice_api_key)


def load() -> Config:
    llm = _choice("VOICE_LLM", "anthropic", LLM_PROVIDERS)
    tts = _choice("VOICE_TTS", "nos", TTS_PROVIDERS)
    config = Config(
        stt=_choice("VOICE_STT", "nos-whisper", STT_PROVIDERS),
        stt_language=_env("VOICE_STT_LANGUAGE", "gl").lower(),
        whisper_model=_env("VOICE_WHISPER_MODEL", "deepdml/faster-whisper-large-v3-turbo-ct2"),
        tts=tts,
        tts_voice=_env("VOICE_TTS_VOICE", DEFAULT_TTS_VOICE[tts]),
        llm=llm,
        llm_model=_env("VOICE_LLM_MODEL", DEFAULT_LLM_MODEL[llm]),
        llm_base_url=_env("VOICE_LLM_BASE_URL"),
        llm_api_key=_secret("VOICE_LLM_API_KEY"),
        elevenlabs_api_key=_secret("ELEVENLABS_API_KEY"),
        deepgram_api_key=_secret("DEEPGRAM_API_KEY"),
        device=_choice("VOICE_DEVICE", "auto", ("auto", "cuda", "cpu")),
        models_dir=Path(_env("VOICE_MODELS_DIR", str(Path.home() / ".cache" / "librequenda-voice"))),
        mcp_url=_env("VOICE_MCP_URL", "http://api:8000/mcp"),
        voice_api_key=_secret("VOICE_API_KEY"),
        public_url=_env("PUBLIC_URL", "http://localhost").rstrip("/"),
        max_call_seconds=int(_env("VOICE_MAX_CALL_SECONDS", "600")),
        max_calls=int(_env("VOICE_MAX_CALLS", "4")),
        turn_silence=float(_env("VOICE_TURN_SILENCE", "0.6")),
        twilio_account_sid=_env("VOICE_TWILIO_ACCOUNT_SID"),
        twilio_auth_token=_secret("VOICE_TWILIO_AUTH_TOKEN"),
        log_level=_env("VOICE_LOG_LEVEL", "INFO").upper(),
    )
    if config.llm == "openai-compatible" and not config.llm_base_url:
        raise ValueError("VOICE_LLM=openai-compatible needs VOICE_LLM_BASE_URL (e.g. http://ollama:11434/v1)")
    if config.llm != "openai-compatible" and not config.llm_api_key:
        raise ValueError(f"VOICE_LLM={config.llm} needs VOICE_LLM_API_KEY (or VOICE_LLM_API_KEY_FILE)")
    if config.stt == "elevenlabs" or config.tts == "elevenlabs":
        if not config.elevenlabs_api_key:
            raise ValueError("ElevenLabs needs ELEVENLABS_API_KEY")
    if config.stt == "deepgram" and not config.deepgram_api_key:
        raise ValueError("VOICE_STT=deepgram needs DEEPGRAM_API_KEY")
    return config


def _self_hosted(url: str) -> bool:
    """A model server on this machine or its private network (Ollama, vLLM…), not a third party."""
    host = urlparse(url).hostname or ""
    if host == "localhost" or "." not in host:  # a compose service name
        return True
    try:
        return ipaddress.ip_address(host).is_private
    except ValueError:
        return False


def third_parties(config: Config) -> dict[str, list[str]]:
    """Who else gets what during a call, for the call panel's privacy note (docs/voice.md)."""
    names = {"elevenlabs": "ElevenLabs", "deepgram": "Deepgram", "anthropic": "Anthropic", "openai": "OpenAI"}
    voice = [names[config.stt]] if config.stt in ("elevenlabs", "deepgram") else []
    text = []
    if config.llm in ("anthropic", "openai"):
        text.append(names[config.llm])
    elif not _self_hosted(config.llm_base_url):
        text.append(urlparse(config.llm_base_url).hostname or config.llm_base_url)
    if config.tts == "elevenlabs" and "ElevenLabs" not in text:
        text.append("ElevenLabs")
    return {"voice": voice, "text": text}
