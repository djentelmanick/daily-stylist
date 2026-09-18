"""Словарь вещей: по нему модель узнаёт фотографию, а форма получает название и категорию.

Список подробнее, чем категории домена: "пуховик" и "ветровка" - обе верхняя одежда,
но по названию видно теплоту, а угадать её отдельным вопросом к модели куда труднее.
"""

from dataclasses import dataclass

SPRING = "spring"
SUMMER = "summer"
AUTUMN = "autumn"
WINTER = "winter"

ALL_SEASONS = (SPRING, SUMMER, AUTUMN, WINTER)

WITHOUT_WARMTH = frozenset({"umbrella", "sunglasses", "bag", "accessory"})

SEASONS_BY_WARMTH: dict[int, tuple[str, ...]] = {
    0: ALL_SEASONS,
    1: (SUMMER,),
    2: (SPRING, SUMMER, AUTUMN),
    3: (SPRING, AUTUMN, WINTER),
    4: (WINTER,),
    5: (WINTER,),
}


@dataclass(frozen=True)
class Label:
    prompt: str
    name: str
    category: str
    warmth: int = 0
    seasons: tuple[str, ...] | None = None
    waterproof: bool = False

    def seasons_of(self) -> tuple[str, ...]:
        if self.seasons is not None:
            return self.seasons
        return SEASONS_BY_WARMTH[self.warmth]


def _label(prompt: str, name: str, category: str, warmth: int = 0, **rest: object) -> Label:
    return Label(prompt=f"a photo of {prompt}", name=name, category=category, warmth=warmth, **rest)  # type: ignore[arg-type]


LABELS: tuple[Label, ...] = (
    _label("a t-shirt", "Футболка", "top", 1),
    _label("a tank top", "Майка", "top", 1),
    _label("a polo shirt", "Поло", "top", 1, seasons=(SPRING, SUMMER)),
    _label("a long sleeve shirt", "Лонгслив", "top", 2),
    _label("a button-up shirt", "Рубашка", "top", 2),
    _label("a blouse", "Блузка", "top", 2),
    _label("a sweatshirt", "Свитшот", "top", 2),
    _label("a hoodie", "Худи", "top", 2),
    _label("a blazer jacket", "Пиджак", "top", 2),
    _label("a knitted sweater", "Свитер", "top", 3),
    _label("a cardigan", "Кардиган", "top", 3),
    _label("a turtleneck sweater", "Водолазка", "top", 3),
    _label("jeans", "Джинсы", "bottom", 2),
    _label("trousers", "Брюки", "bottom", 2),
    _label("chino pants", "Чиносы", "bottom", 2),
    _label("sweatpants", "Спортивные штаны", "bottom", 2),
    _label("leggings", "Леггинсы", "bottom", 2),
    _label("a skirt", "Юбка", "bottom", 2),
    _label("shorts", "Шорты", "bottom", 1),
    _label("a dress", "Платье", "dress", 2),
    _label("a summer sundress", "Сарафан", "dress", 1),
    _label("a knitted dress", "Вязаное платье", "dress", 3),
    _label("a jumpsuit", "Комбинезон", "jumpsuit", 2),
    _label("dungaree overalls", "Комбинезон", "jumpsuit", 2),
    _label("a denim jacket", "Джинсовая куртка", "outerwear", 2),
    _label("a leather jacket", "Кожаная куртка", "outerwear", 2),
    _label("a bomber jacket", "Бомбер", "outerwear", 2),
    _label("a windbreaker jacket", "Ветровка", "outerwear", 2),
    _label("a raincoat", "Дождевик", "outerwear", 2, waterproof=True),
    _label("a trench coat", "Тренч", "outerwear", 3),
    _label("a wool coat", "Пальто", "outerwear", 3),
    _label("a fleece jacket", "Флиска", "outerwear", 3),
    _label("a parka", "Парка", "outerwear", 4),
    _label("a winter jacket", "Зимняя куртка", "outerwear", 4),
    _label("a puffer down jacket", "Пуховик", "outerwear", 5),
    _label("a fur coat", "Шуба", "outerwear", 5),
    _label("sneakers", "Кроссовки", "shoes", 2),
    _label("running shoes", "Кроссовки", "shoes", 2),
    _label("loafers", "Лоферы", "shoes", 2),
    _label("dress shoes", "Туфли", "shoes", 2),
    _label("high heel shoes", "Туфли на каблуке", "shoes", 2),
    _label("ballet flats", "Балетки", "shoes", 2),
    _label("sandals", "Сандалии", "shoes", 1),
    _label("flip flops", "Шлёпанцы", "shoes", 1),
    _label("leather boots", "Ботинки", "shoes", 3),
    _label("ankle boots", "Ботильоны", "shoes", 3),
    _label("rubber rain boots", "Резиновые сапоги", "shoes", 3, waterproof=True),
    _label("winter snow boots", "Зимние ботинки", "shoes", 4),
    _label("ugg boots", "Угги", "shoes", 4),
    _label("socks", "Носки", "socks", 1, seasons=ALL_SEASONS),
    _label("wool socks", "Шерстяные носки", "socks", 3),
    _label("tights", "Колготки", "socks", 2, seasons=(SPRING, AUTUMN, WINTER)),
    _label("a baseball cap", "Кепка", "hat", 1, seasons=(SPRING, SUMMER, AUTUMN)),
    _label("a bucket hat", "Панама", "hat", 1),
    _label("a straw sun hat", "Шляпа", "hat", 1),
    _label("a knitted beanie", "Шапка", "hat", 4),
    _label("a fur winter hat", "Меховая шапка", "hat", 5),
    _label("a silk neck scarf", "Платок", "scarf", 1, seasons=ALL_SEASONS),
    _label("a knitted scarf", "Шарф", "scarf", 3),
    _label("a wool snood", "Снуд", "scarf", 4),
    _label("thermal underwear", "Термобельё", "thermal", 4),
    _label("an umbrella", "Зонт", "umbrella", seasons=(SPRING, SUMMER, AUTUMN), waterproof=True),
    _label("sunglasses", "Солнцезащитные очки", "sunglasses", seasons=(SPRING, SUMMER)),
    _label("a backpack", "Рюкзак", "bag"),
    _label("a handbag", "Сумка", "bag"),
    _label("a tote bag", "Сумка-шоппер", "bag"),
    _label("a crossbody bag", "Сумка через плечо", "bag"),
    _label("a leather belt", "Ремень", "accessory"),
    _label("gloves", "Перчатки", "accessory", seasons=(AUTUMN, WINTER)),
    _label("a wristwatch", "Часы", "accessory"),
)

PROMPTS: tuple[str, ...] = tuple(label.prompt for label in LABELS)
