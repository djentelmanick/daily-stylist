import pytest

from app.recognize.labels import LABELS, SEASONS_BY_WARMTH, WITHOUT_WARMTH
from app.recognize.scoring import choose, to_recognition

THRESHOLD = 0.35


def scores_for(names: dict[str, float]) -> list[float]:
    return [names.get(label.name, 0.0) for label in LABELS]


def test_уверенная_подпись_выбирается():
    label = choose(scores_for({"Пуховик": 12.0}), LABELS, THRESHOLD)

    assert label is not None
    assert label.name == "Пуховик"
    assert label.category == "outerwear"
    assert label.warmth == 5


def test_без_уверенности_подсказки_нет():
    assert choose([0.0] * len(LABELS), LABELS, THRESHOLD) is None


def test_подписи_одной_категории_складываются():
    scores = scores_for({"Кроссовки": 6.0})
    single = scores_for({"Лоферы": 6.0})

    assert choose(single, LABELS, 0.9) is None
    assert choose(scores, LABELS, 0.9) is not None


def test_у_аксессуаров_нет_теплоты():
    for label in LABELS:
        assert (label.category in WITHOUT_WARMTH) == (label.warmth == 0), label.name


def test_сезоны_берутся_из_теплоты_если_не_заданы():
    label = next(label for label in LABELS if label.name == "Свитер")

    assert label.seasons_of() == SEASONS_BY_WARMTH[3]


def test_подсказка_собирается_из_подписи_и_цветов():
    label = choose(scores_for({"Дождевик": 12.0}), LABELS, THRESHOLD)
    assert label is not None

    recognition = to_recognition(label, "navy", ["white", "gray"])

    assert recognition.name == "Дождевик"
    assert recognition.category == "outerwear"
    assert recognition.main_color == "navy"
    assert recognition.extra_colors == ["white", "gray"]
    assert recognition.warmth_level == 2
    assert recognition.waterproof is True


@pytest.mark.parametrize("warmth", sorted(SEASONS_BY_WARMTH))
def test_у_каждой_теплоты_есть_сезон(warmth: int):
    assert SEASONS_BY_WARMTH[warmth]
