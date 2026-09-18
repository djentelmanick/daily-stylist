import numpy as np
from PIL import Image

# Палитра домена. Значения совпадают с образцами цвета в Mini App.
PALETTE: dict[str, tuple[int, int, int]] = {
    "red": (217, 48, 37),
    "burgundy": (128, 0, 32),
    "blue": (26, 86, 219),
    "light_blue": (142, 205, 247),
    "navy": (31, 42, 68),
    "green": (46, 125, 50),
    "khaki": (168, 159, 104),
    "yellow": (242, 194, 0),
    "black": (17, 17, 17),
    "white": (255, 255, 255),
    "gray": (158, 158, 158),
    "brown": (109, 76, 65),
    "beige": (232, 217, 181),
    "orange": (245, 124, 0),
    "purple": (123, 31, 162),
    "pink": (244, 143, 177),
}

SIZE = 160
CLUSTERS = 4
# Доля кадра от края, по которой угадывается фон, и от центра, где ищется вещь.
BORDER = 0.08
CENTER = 0.62
# Расстояние в Lab, ближе которого цвет считается тем же самым.
BACKGROUND_DISTANCE = 14.0
EXTRA_SHARE = 0.12
MAX_EXTRA = 3


def colors_of(image: Image.Image) -> tuple[str, list[str]]:
    small = image.convert("RGB").resize((SIZE, SIZE), Image.Resampling.BILINEAR)
    lab = to_lab(np.asarray(small, dtype=np.float64) / 255.0)

    background = border_color(lab)
    centers, shares = cluster(crop_center(lab).reshape(-1, 3), CLUSTERS)

    return pick(centers, shares, background)


def pick(centers: np.ndarray, shares: np.ndarray, background: np.ndarray) -> tuple[str, list[str]]:
    foreground = [
        index
        for index in range(len(centers))
        if float(np.linalg.norm(centers[index] - background)) > BACKGROUND_DISTANCE
    ]
    # Вещь сняли на своём же фоне - тогда фон и есть вещь.
    if not foreground:
        foreground = list(range(len(centers)))

    weights: dict[str, float] = {}
    for index in foreground:
        name = nearest_color(centers[index])
        weights[name] = weights.get(name, 0.0) + float(shares[index])

    order = sorted(weights, key=lambda name: weights[name], reverse=True)
    main = order[0]
    extras = [name for name in order[1:] if weights[name] >= EXTRA_SHARE][:MAX_EXTRA]
    return main, extras


def nearest_color(lab: np.ndarray) -> str:
    return min(PALETTE_LAB, key=lambda name: float(np.linalg.norm(PALETTE_LAB[name] - lab)))


def cluster(points: np.ndarray, count: int, iterations: int = 12) -> tuple[np.ndarray, np.ndarray]:
    """k-means с постоянным зерном: одна и та же фотография должна давать один и тот же ответ."""
    rng = np.random.default_rng(0)
    centers = points[rng.choice(len(points), size=count, replace=False)].copy()

    labels = np.zeros(len(points), dtype=int)
    for _ in range(iterations):
        distances = ((points[:, None, :] - centers[None, :, :]) ** 2).sum(axis=2)
        labels = distances.argmin(axis=1)
        for index in range(count):
            chosen = points[labels == index]
            if len(chosen) > 0:
                centers[index] = chosen.mean(axis=0)

    shares = np.array([float((labels == index).mean()) for index in range(count)])
    return centers, shares


def border_color(lab: np.ndarray) -> np.ndarray:
    width = max(1, int(lab.shape[0] * BORDER))
    ring = np.concatenate(
        [
            lab[:width].reshape(-1, 3),
            lab[-width:].reshape(-1, 3),
            lab[:, :width].reshape(-1, 3),
            lab[:, -width:].reshape(-1, 3),
        ]
    )
    return np.median(ring, axis=0)


def crop_center(lab: np.ndarray) -> np.ndarray:
    margin = int(lab.shape[0] * (1 - CENTER) / 2)
    return lab[margin : lab.shape[0] - margin, margin : lab.shape[1] - margin]


def to_lab(rgb: np.ndarray) -> np.ndarray:
    linear = np.where(rgb <= 0.04045, rgb / 12.92, ((rgb + 0.055) / 1.055) ** 2.4)
    matrix = np.array(
        [
            [0.4124564, 0.3575761, 0.1804375],
            [0.2126729, 0.7151522, 0.0721750],
            [0.0193339, 0.1191920, 0.9503041],
        ]
    )
    xyz = linear @ matrix.T / np.array([0.95047, 1.0, 1.08883])
    f = np.where(xyz > 0.008856, np.cbrt(xyz), 7.787 * xyz + 16 / 116)
    return np.stack(
        [116 * f[..., 1] - 16, 500 * (f[..., 0] - f[..., 1]), 200 * (f[..., 1] - f[..., 2])],
        axis=-1,
    )


PALETTE_LAB = {name: to_lab(np.array(rgb, dtype=np.float64) / 255.0) for name, rgb in PALETTE.items()}
