import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    listen_addr: str
    model_path: str
    prompts_path: str
    threshold: float
    max_photo_bytes: int
    workers: int


def load() -> Settings:
    return Settings(
        listen_addr=os.getenv("VISION_LISTEN_ADDR", "0.0.0.0:9090"),
        model_path=os.getenv("VISION_MODEL_PATH", "/models/vision_model.onnx"),
        prompts_path=os.getenv("VISION_PROMPTS_PATH", "/models/prompts.npy"),
        threshold=float(os.getenv("VISION_THRESHOLD", "0.35")),
        max_photo_bytes=int(os.getenv("VISION_MAX_PHOTO_BYTES", str(20 << 20))),
        workers=int(os.getenv("VISION_WORKERS", "2")),
    )
