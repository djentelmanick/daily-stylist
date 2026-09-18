from dataclasses import dataclass, field


@dataclass(frozen=True)
class Recognition:
    """Подсказки для формы. Пустое поле значит "не знаю": форма его не трогает."""

    name: str = ""
    category: str = ""
    main_color: str = ""
    extra_colors: list[str] = field(default_factory=list)
    seasons: list[str] = field(default_factory=list)
    warmth_level: int = 0
    waterproof: bool = False
