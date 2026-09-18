import io

import numpy as np
from PIL import Image

from app.recognize.local import IMAGE_SIZE, MEAN, STD, open_image, preprocess


def test_картинка_приводится_к_входу_модели():
    pixels = preprocess(Image.new("RGB", (900, 1200), (255, 255, 255)))

    assert pixels.shape == (1, 3, IMAGE_SIZE, IMAGE_SIZE)
    assert pixels.dtype == np.float32
    assert np.allclose(pixels[0, :, 0, 0], (1.0 - MEAN) / STD)


def test_вырезается_середина_а_не_край():
    photo = Image.new("RGB", (300, 900), (0, 0, 0))
    photo.paste(Image.new("RGB", (300, 300), (255, 255, 255)), (0, 300))

    pixels = preprocess(photo)

    white = (1.0 - MEAN) / STD
    assert np.allclose(pixels[0, :, IMAGE_SIZE // 2, IMAGE_SIZE // 2], white, atol=1e-3)


def test_снимок_с_телефона_поворачивается_по_exif():
    photo = Image.new("RGB", (400, 200), (255, 0, 0))
    exif = photo.getexif()
    exif[0x0112] = 6
    encoded = io.BytesIO()
    photo.save(encoded, format="JPEG", exif=exif)

    assert open_image(encoded.getvalue()).size == (200, 400)
