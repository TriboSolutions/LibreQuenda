"""Pipecat services for Proxecto Nós' Galician models (Apache-2.0), running on this server."""

import asyncio
import re
from collections.abc import AsyncGenerator

import numpy as np
from loguru import logger
from pipecat.audio.utils import create_stream_resampler
from pipecat.frames.frames import ErrorFrame, Frame, TranscriptionFrame, TTSAudioRawFrame
from pipecat.services.settings import STTSettings, TTSSettings
from pipecat.services.stt_service import SegmentedSTTService
from pipecat.services.tts_service import TTSService
from pipecat.services.whisper.stt import WhisperSTTService
from pipecat.transcriptions.language import Language
from pipecat.utils.time import time_now_iso8601

from lqvoice import models

# punctuation Whisper sometimes puts before the first word
LEADING_MARKS = re.compile(r"^[\s,.;:…-]+")

# Nós' voices are trained at 16 kHz
NOS_SAMPLE_RATE = 16000


class SharedWhisperSTTService(WhisperSTTService):
    """Whisper on faster-whisper (Nós' Galician fine-tune by default), loaded once per process for every call."""

    def __init__(self, *, model_path: str, device: str, **kwargs):
        self._model_path = model_path
        super().__init__(device=device, **kwargs)

    def _load(self):
        self._model = models.whisper(self._model_path, self._device).model

    async def run_stt(self, audio: bytes) -> AsyncGenerator[Frame, None]:
        async for frame in super().run_stt(audio):
            # Whisper sometimes opens with a stray mark (", Ola…"): the captions show it
            if isinstance(frame, TranscriptionFrame):
                frame.text = LEADING_MARKS.sub("", frame.text)
                if not frame.text:
                    continue
            yield frame


class NosWav2Vec2STTService(SegmentedSTTService):
    """Nós' wav2vec2 XLS-R 300M for Galician: small enough for a CPU."""

    def __init__(self, *, model_path: str, device: str, **kwargs):
        super().__init__(settings=STTSettings(model=model_path, language=Language.GL), **kwargs)
        self._device = device
        self._shared = models.wav2vec2(model_path, device)

    @property
    def wants_wav_segments(self) -> bool:
        return False

    def can_generate_metrics(self) -> bool:
        return True

    def _transcribe(self, audio: np.ndarray) -> str:
        import torch

        processor, model = self._shared.model
        with self._shared.lock, torch.inference_mode():
            inputs = processor(audio, sampling_rate=16000, return_tensors="pt").to(self._device)
            ids = model(**inputs).logits.argmax(-1)
        return processor.batch_decode(ids)[0].strip()

    async def run_stt(self, audio: bytes) -> AsyncGenerator[Frame, None]:
        await self.start_processing_metrics()
        samples = np.frombuffer(audio, dtype=np.int16).astype(np.float32) / 32768.0
        if self.sample_rate != 16000:
            import soxr

            samples = soxr.resample(samples, self.sample_rate, 16000)
        try:
            text = await asyncio.to_thread(self._transcribe, samples)
        except Exception as e:
            yield ErrorFrame(error=f"wav2vec2 failed: {e}")
            return
        finally:
            await self.stop_processing_metrics()
        if text:
            yield TranscriptionFrame(text, self._user_id, time_now_iso8601(), Language.GL)


def clauses(text: str, longest: int = 90) -> list[str]:
    """A long sentence in pieces at its commas, so the first one plays while the next is synthesised."""
    if len(text) <= longest:
        return [text]
    pieces, current = [], ""
    for part in re.split(r"(?<=[,;:])\s+", text):
        if current and len(current) + len(part) > longest:
            pieces.append(current)
            current = part
        else:
            current = f"{current} {part}".strip()
    return [*pieces, current] if current else pieces


class NosTTSService(TTSService):
    """Nós' VITS voices (graphemes: no phonetic front end), sentence by sentence as the agent writes."""

    def __init__(self, *, checkpoint: str, config: str, voice: str, device: str, **kwargs):
        super().__init__(
            push_start_frame=True,
            push_stop_frames=True,
            # a long sentence on a CPU takes seconds: not a silent TTS
            stop_frame_timeout_s=10.0,
            settings=TTSSettings(model=None, voice=voice, language=Language.GL),
            **kwargs,
        )
        self._shared = models.vits(checkpoint, config, device)
        self._resampler = create_stream_resampler()

    def can_generate_metrics(self) -> bool:
        return True

    def _synthesize(self, text: str) -> bytes:
        with self._shared.lock:
            wav = self._shared.model.tts(text)
        return (np.clip(np.asarray(wav, dtype=np.float32), -1, 1) * 32767).astype(np.int16).tobytes()

    async def run_tts(self, text: str, context_id: str) -> AsyncGenerator[Frame, None]:
        if not text.strip():
            return
        try:
            await self.start_tts_usage_metrics(text)
            for piece in clauses(text):
                audio = await asyncio.to_thread(self._synthesize, piece)
                await self.stop_ttfb_metrics()
                audio = await self._resampler.resample(audio, NOS_SAMPLE_RATE, self.sample_rate)
                yield TTSAudioRawFrame(audio=audio, sample_rate=self.sample_rate, num_channels=1, context_id=context_id)
        except Exception as e:
            logger.exception("Nós TTS failed")
            yield ErrorFrame(error=f"Nós TTS failed: {e}")
        finally:
            await self.stop_ttfb_metrics()
