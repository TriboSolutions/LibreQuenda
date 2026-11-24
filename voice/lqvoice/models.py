"""Local model weights: downloaded on first start into VOICE_MODELS_DIR and loaded once per process.

Every call shares the loaded models; a lock per model keeps two calls from running the
same model at once (the GPU is shared with other work and a voice line has few calls).
"""

import shutil
import threading
from functools import cache
from pathlib import Path

from loguru import logger

NOS_WHISPER = "proxectonos/whisper-large-v3-turbo-gl-v1.0"
NOS_WAV2VEC2 = "proxectonos/Nos_ASR-wav2vec2-xls-r-300m-gl"
NOS_VOICES = {
    "brais": ("proxectonos/Nos_TTS-brais-vits-graphemes", "brais.pth"),
    "celtia": ("proxectonos/Nos_TTS-celtia-vits-graphemes", "celtia.pth"),
}


def device(preferred: str) -> str:
    if preferred != "auto":
        return preferred
    import torch

    return "cuda" if torch.cuda.is_available() else "cpu"


def _download(repo: str, models_dir: Path, patterns: list[str]) -> Path:
    from huggingface_hub import snapshot_download

    return Path(snapshot_download(repo, cache_dir=models_dir / "hub", allow_patterns=patterns))


def nos_whisper_ct2(models_dir: Path) -> Path:
    """Nós' Galician Whisper as CTranslate2 (faster-whisper), converted once from the Hugging Face weights."""
    target = models_dir / "ct2" / NOS_WHISPER.split("/")[1]
    if (target / "model.bin").exists():
        return target
    source = _download(NOS_WHISPER, models_dir, ["*.json", "model.safetensors", "merges.txt", "vocab.json"])
    logger.info(f"converting {NOS_WHISPER} to CTranslate2 (once)")
    partial = target.with_name(target.name + ".partial")
    shutil.rmtree(partial, ignore_errors=True)
    partial.parent.mkdir(parents=True, exist_ok=True)
    from ctranslate2.converters import TransformersConverter

    converter = TransformersConverter(str(source), copy_files=["tokenizer.json", "preprocessor_config.json"])
    converter.convert(str(partial), quantization="float16")
    partial.rename(target)
    return target


def nos_wav2vec2(models_dir: Path) -> Path:
    return _download(NOS_WAV2VEC2, models_dir, ["*.json", "model.safetensors"])


def nos_voice(models_dir: Path, voice: str) -> tuple[Path, Path]:
    if voice not in NOS_VOICES:
        raise ValueError(f"Nós voice must be one of {', '.join(NOS_VOICES)}, got '{voice}'")
    repo, checkpoint = NOS_VOICES[voice]
    folder = _download(repo, models_dir, ["config.json", checkpoint])
    return folder / checkpoint, folder / "config.json"


class Shared:
    """A loaded model and the lock that serialises its use."""

    def __init__(self, model):
        self.model = model
        self.lock = threading.Lock()


@cache
def whisper(path: str, dev: str) -> Shared:
    from faster_whisper import WhisperModel

    compute = "float16" if dev == "cuda" else "int8"
    logger.info(f"loading Whisper {path} on {dev} ({compute})")
    return Shared(WhisperModel(path, device=dev, compute_type=compute))


@cache
def wav2vec2(path: str, dev: str) -> Shared:
    from transformers import AutoModelForCTC, AutoProcessor

    logger.info(f"loading wav2vec2 {path} on {dev}")
    processor = AutoProcessor.from_pretrained(path)
    model = AutoModelForCTC.from_pretrained(path).to(dev).eval()
    return Shared((processor, model))


@cache
def vits(checkpoint: str, config: str, dev: str) -> Shared:
    from TTS.utils.synthesizer import Synthesizer

    logger.info(f"loading Nós voice {checkpoint} on {dev}")
    synthesizer = Synthesizer(tts_checkpoint=checkpoint, tts_config_path=config, use_cuda=False)
    synthesizer.tts_model.to(dev)
    return Shared(synthesizer)
