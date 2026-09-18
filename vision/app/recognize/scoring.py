from collections.abc import Sequence

import numpy as np

from .labels import Label
from .recognition import Recognition


def choose(scores: Sequence[float], labels: Sequence[Label], threshold: float) -> Label | None:
    """Уверенность считается по категории, а не по ярлыку.

    У одной категории несколько подписей ("кроссовки" и "кеды"), и близкие ярлыки делят
    уверенность между собой. Сложив их, мы отличаем "не понял, что это" от "не понял,
    как это назвать": во втором случае категорию подсказать всё равно стоит.
    """
    probabilities = softmax(np.asarray(scores, dtype=np.float64))

    weights: dict[str, float] = {}
    for label, probability in zip(labels, probabilities, strict=True):
        weights[label.category] = weights.get(label.category, 0.0) + float(probability)

    category = max(weights, key=lambda name: weights[name])
    if weights[category] < threshold:
        return None

    best = max(
        (index for index, label in enumerate(labels) if label.category == category),
        key=lambda index: probabilities[index],
    )
    return labels[best]


def softmax(scores: np.ndarray) -> np.ndarray:
    shifted = np.exp(scores - scores.max())
    return shifted / shifted.sum()


def to_recognition(label: Label, main_color: str, extra_colors: Sequence[str]) -> Recognition:
    return Recognition(
        name=label.name,
        category=label.category,
        main_color=main_color,
        extra_colors=list(extra_colors),
        seasons=list(label.seasons_of()),
        warmth_level=label.warmth,
        waterproof=label.waterproof,
    )
