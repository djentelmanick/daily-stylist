import logging
import signal
from concurrent import futures

import grpc

from . import config, recognize
from .generated import vision_pb2_grpc
from .server import RecognizerService

logger = logging.getLogger(__name__)

SHUTDOWN_GRACE_SECONDS = 10
MESSAGE_MARGIN_BYTES = 1 << 20


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    settings = config.load()

    # Распознаватель готовится до старта сервера: пока он не готов, отвечать нечем,
    # и порт лучше не открывать - по нему же docker понимает, что сервис готов.
    recognizer = recognize.build(settings)

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=settings.workers),
        options=[("grpc.max_receive_message_length", settings.max_photo_bytes + MESSAGE_MARGIN_BYTES)],
    )
    vision_pb2_grpc.add_RecognizerServicer_to_server(RecognizerService(recognizer), server)
    server.add_insecure_port(settings.listen_addr)
    server.start()
    logger.info("слушаю %s", settings.listen_addr)

    stop = lambda *_: server.stop(SHUTDOWN_GRACE_SECONDS)  # noqa: E731
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)

    server.wait_for_termination()
    logger.info("остановлен")


if __name__ == "__main__":
    main()
