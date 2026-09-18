"""Проверка для docker: порт открыт - значит, модель загружена и сервис готов отвечать."""

import os
import sys

import grpc

listen = os.getenv("VISION_LISTEN_ADDR", "0.0.0.0:9090")
address = listen.replace("0.0.0.0", "localhost").replace("[::]", "localhost")

try:
    grpc.channel_ready_future(grpc.insecure_channel(address)).result(timeout=3)
except grpc.FutureTimeoutError:
    sys.exit(1)
