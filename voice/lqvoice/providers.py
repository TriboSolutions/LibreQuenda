"""STT, TTS and LLM services by configuration: the open, self-hosted models by default, any other as an adapter."""

import aiohttp
from pipecat.transcriptions.language import Language
from pipecat.utils.text.transforms import strip_markdown

from lqvoice import models
from lqvoice.config import Config
from lqvoice.normalize import tts_transform
from lqvoice.nos import NosTTSService, NosWav2Vec2STTService, SharedWhisperSTTService

GALICIAN_TRANSFORMS = [("*", strip_markdown), ("*", tts_transform)]


def stt_language(config: Config) -> Language | None:
    return None if config.stt_language == "auto" else Language(config.stt_language)


def warm_up(config: Config) -> None:
    """Download and load the local models before the first call, so it does not wait for them."""
    dev = models.device(config.device)
    if config.stt == "nos-whisper":
        models.whisper(str(models.nos_whisper_ct2(config.models_dir)), dev)
    elif config.stt == "nos-wav2vec2":
        models.wav2vec2(str(models.nos_wav2vec2(config.models_dir)), dev)
    elif config.stt == "whisper":
        models.whisper(config.whisper_model, dev)
    if config.tts == "nos":
        checkpoint, voice_config = models.nos_voice(config.models_dir, config.tts_voice)
        # one sentence now: the first call does not wait for the compilers' first run (numba, CUDA)
        models.vits(str(checkpoint), str(voice_config), dev).model.tts("Ola.")


def make_stt(config: Config, http: aiohttp.ClientSession, hotwords: str | None = None):
    dev = models.device(config.device)
    if config.stt in ("nos-whisper", "whisper"):
        path = str(models.nos_whisper_ct2(config.models_dir)) if config.stt == "nos-whisper" else config.whisper_model
        return SharedWhisperSTTService(
            model_path=path,
            device=dev,
            settings=SharedWhisperSTTService.Settings(
                model=path, language=stt_language(config), initial_prompt=hotwords
            ),
        )
    if config.stt == "nos-wav2vec2":
        return NosWav2Vec2STTService(model_path=str(models.nos_wav2vec2(config.models_dir)), device=dev)
    if config.stt == "elevenlabs":
        from pipecat.services.elevenlabs.stt import ElevenLabsSTTService

        return ElevenLabsSTTService(
            api_key=config.elevenlabs_api_key,
            aiohttp_session=http,
            settings=ElevenLabsSTTService.Settings(language=stt_language(config) or Language.GL),
        )
    from pipecat.services.deepgram.stt import DeepgramSTTService

    # Deepgram has no Galician: Spanish, or its multilingual mode
    language = "multi" if config.stt_language in ("gl", "auto") else config.stt_language
    return DeepgramSTTService(api_key=config.deepgram_api_key, settings=DeepgramSTTService.Settings(language=language))


def make_tts(config: Config):
    if config.tts == "nos":
        dev = models.device(config.device)
        checkpoint, voice_config = models.nos_voice(config.models_dir, config.tts_voice)
        return NosTTSService(
            checkpoint=str(checkpoint),
            config=str(voice_config),
            voice=config.tts_voice,
            device=dev,
            text_transforms=GALICIAN_TRANSFORMS,
        )
    if config.tts == "piper":
        from pipecat.services.piper.tts import PiperTTSService

        return PiperTTSService(
            download_dir=config.models_dir / "piper",
            use_cuda=models.device(config.device) == "cuda",
            settings=PiperTTSService.Settings(voice=config.tts_voice),
            text_transforms=GALICIAN_TRANSFORMS,
        )
    from pipecat.services.elevenlabs.tts import ElevenLabsTTSService

    return ElevenLabsTTSService(
        api_key=config.elevenlabs_api_key,
        settings=ElevenLabsTTSService.Settings(voice=config.tts_voice, language=Language.GL),
        text_transforms=GALICIAN_TRANSFORMS,
    )


def make_llm(config: Config, system: str):
    if config.llm == "anthropic":
        from anthropic import AsyncAnthropic
        from pipecat.services.anthropic.llm import AnthropicLLMService

        # Pipecat sends each Claude model's lowest thinking setting unless told otherwise: what a voice wants
        client = AsyncAnthropic(api_key=config.llm_api_key, base_url=config.llm_base_url or None)
        return AnthropicLLMService(
            api_key=config.llm_api_key,
            client=client,
            settings=AnthropicLLMService.Settings(model=config.llm_model, system_instruction=system, max_tokens=1024),
        )
    from pipecat.services.openai.llm import OpenAILLMService

    return OpenAILLMService(
        api_key=config.llm_api_key or "none",
        base_url=config.llm_base_url or None,
        settings=OpenAILLMService.Settings(model=config.llm_model, system_instruction=system),
    )
