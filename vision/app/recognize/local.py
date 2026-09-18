"""Подписи посчитаны при сборке образа, поэтому здесь нет ни текстовой части модели, ни токенизатора."""

import io
import logging

import numpy as np
import onnxruntime
from PIL import Image, ImageOps

from .labels import LABELS
from .palette import colors_of
from .recognition import Recognition
from .scoring import choose, to_recognition

logger = logging.getLogger(__name__)

IMAGE_SIZE = 224
MEAN = np.array([0.48145466, 0.4578275, 0.40821073], dtype=np.float32)
STD = np.array([0.26862954, 0.26130258, 0.27577711], dtype=np.float32)
# Масштаб, с которым CLIP учили: без него похожести не превращаются в вероятности.
LOGIT_SCALE = 100.0


class LocalRecognizer:
    def __init__(self, model_path: str, prompts_path: str, threshold: float) -> None:
        self._threshold = threshold
        self._prompts = np.load(prompts_path)
        if len(self._prompts) != len(LABELS):
            raise ValueError(f"подписей {len(LABELS)}, а посчитано {len(self._prompts)}: пересоберите образ")
        self._session = onnxruntime.InferenceSession(model_path, providers=["CPUExecutionProvider"])
        logger.info("модель готова, подписей: %d", len(LABELS))

    def recognize(self, photo: bytes, content_type: str) -> Recognition:
        image = open_image(photo)

        label = choose(self._similarity(image), LABELS, self._threshold)
        if label is None:
            return Recognition()

        main_color, extra_colors = colors_of(image)
        return to_recognition(label, main_color, extra_colors)

    def _similarity(self, image: Image.Image) -> list[float]:
        (features,) = self._session.run(None, {"pixel_values": preprocess(image)})
        features = features / np.linalg.norm(features, axis=-1, keepdims=True)
        return (LOGIT_SCALE * features @ self._prompts.T)[0].tolist()


def preprocess(image: Image.Image) -> np.ndarray:
    """Та же подготовка, на которой модель учили: иначе похожести поедут."""
    width, height = image.size
    scale = IMAGE_SIZE / min(width, height)
    resized = image.resize((round(width * scale), round(height * scale)), Image.Resampling.BICUBIC)

    left = (resized.width - IMAGE_SIZE) // 2
    top = (resized.height - IMAGE_SIZE) // 2
    cropped = resized.crop((left, top, left + IMAGE_SIZE, top + IMAGE_SIZE))

    pixels = np.asarray(cropped, dtype=np.float32) / 255.0
    return ((pixels - MEAN) / STD).transpose(2, 0, 1)[None]


def open_image(photo: bytes) -> Image.Image:
    # Поворот из EXIF: снимок с телефона иначе окажется на боку.
    return ImageOps.exif_transpose(Image.open(io.BytesIO(photo))).convert("RGB")
