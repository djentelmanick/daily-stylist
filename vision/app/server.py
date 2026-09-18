import logging

import grpc

from .generated import vision_pb2, vision_pb2_grpc
from .recognize import Recognizer

logger = logging.getLogger(__name__)


class RecognizerService(vision_pb2_grpc.RecognizerServicer):
    def __init__(self, recognizer: Recognizer) -> None:
        self._recognizer = recognizer

    def RecognizeItem(  # noqa: N802 - имя метода задано контрактом
        self,
        request: vision_pb2.RecognizeItemRequest,
        context: grpc.ServicerContext,
    ) -> vision_pb2.RecognizeItemResponse:
        if not request.photo:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "пустая фотография")

        try:
            result = self._recognizer.recognize(request.photo, request.content_type)
        except Exception:
            # Что именно сломалось, знает лог; наружу уходит только код: подсказки не будет.
            logger.exception("распознавание не удалось")
            context.abort(grpc.StatusCode.INTERNAL, "не удалось распознать фотографию")

        logger.info("распознано: %s / %s", result.category or "-", result.name or "-")
        return vision_pb2.RecognizeItemResponse(
            name=result.name,
            category=result.category,
            main_color=result.main_color,
            extra_colors=result.extra_colors,
            seasons=result.seasons,
            warmth_level=result.warmth_level,
            waterproof=result.waterproof,
        )
