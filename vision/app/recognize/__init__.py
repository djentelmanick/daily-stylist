from typing import Protocol

from ..config import Settings
from .recognition import Recognition


class Recognizer(Protocol):
    def recognize(self, photo: bytes, content_type: str) -> Recognition: ...


def build(settings: Settings) -> Recognizer:
    from .local import LocalRecognizer

    return LocalRecognizer(
        model_path=settings.model_path,
        prompts_path=settings.prompts_path,
        threshold=settings.threshold,
    )


__all__ = ["Recognition", "Recognizer", "build"]
