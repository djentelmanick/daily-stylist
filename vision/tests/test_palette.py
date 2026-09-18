import numpy as np
from PIL import Image

from app.recognize.palette import PALETTE, colors_of, nearest_color, to_lab


def image_of(color: tuple[int, int, int], size: tuple[int, int] = (600, 800)) -> Image.Image:
    return Image.new("RGB", size, color)


def test_каждый_цвет_палитры_узнаётся_в_себе():
    for name, rgb in PALETTE.items():
        assert colors_of(image_of(rgb))[0] == name


def test_фон_не_становится_цветом_вещи():
    photo = image_of((255, 255, 255))
    photo.paste(image_of((217, 48, 37), (300, 400)), (150, 200))

    assert colors_of(photo) == ("red", [])


def test_второй_цвет_попадает_в_дополнительные():
    photo = image_of((255, 255, 255))
    photo.paste(image_of((26, 86, 219), (300, 400)), (150, 200))
    photo.paste(image_of((242, 194, 0), (300, 130)), (150, 200))

    main, extras = colors_of(photo)

    assert main == "blue"
    assert extras == ["yellow"]


def test_вещь_на_своём_же_фоне_всё_равно_даёт_цвет():
    assert colors_of(image_of((17, 17, 17))) == ("black", [])


def test_тёмно_красный_ближе_к_бордовому_чем_к_красному():
    assert nearest_color(to_lab(np.array((120, 20, 20), dtype=float) / 255.0)) == "burgundy"
