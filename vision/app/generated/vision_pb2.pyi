from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class RecognizeItemRequest(_message.Message):
    __slots__ = ("photo", "content_type")
    PHOTO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    photo: bytes
    content_type: str
    def __init__(self, photo: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class RecognizeItemResponse(_message.Message):
    __slots__ = ("name", "category", "main_color", "extra_colors", "seasons", "warmth_level", "waterproof")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    MAIN_COLOR_FIELD_NUMBER: _ClassVar[int]
    EXTRA_COLORS_FIELD_NUMBER: _ClassVar[int]
    SEASONS_FIELD_NUMBER: _ClassVar[int]
    WARMTH_LEVEL_FIELD_NUMBER: _ClassVar[int]
    WATERPROOF_FIELD_NUMBER: _ClassVar[int]
    name: str
    category: str
    main_color: str
    extra_colors: _containers.RepeatedScalarFieldContainer[str]
    seasons: _containers.RepeatedScalarFieldContainer[str]
    warmth_level: int
    waterproof: bool
    def __init__(self, name: _Optional[str] = ..., category: _Optional[str] = ..., main_color: _Optional[str] = ..., extra_colors: _Optional[_Iterable[str]] = ..., seasons: _Optional[_Iterable[str]] = ..., warmth_level: _Optional[int] = ..., waterproof: _Optional[bool] = ...) -> None: ...
