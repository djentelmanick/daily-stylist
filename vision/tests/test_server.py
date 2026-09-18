from concurrent import futures

import grpc
import pytest

from app.generated import vision_pb2, vision_pb2_grpc
from app.recognize import Recognition
from app.server import RecognizerService


class FakeRecognizer:
    def __init__(self, result: Recognition | Exception) -> None:
        self.result = result
        self.seen: bytes = b""

    def recognize(self, photo: bytes, content_type: str) -> Recognition:
        self.seen = photo
        if isinstance(self.result, Exception):
            raise self.result
        return self.result


@pytest.fixture
def client():
    def serve(recognizer):
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))
        vision_pb2_grpc.add_RecognizerServicer_to_server(RecognizerService(recognizer), server)
        port = server.add_insecure_port("localhost:0")
        server.start()
        stub = vision_pb2_grpc.RecognizerStub(grpc.insecure_channel(f"localhost:{port}"))
        servers.append(server)
        return stub

    servers: list[grpc.Server] = []
    yield serve
    for server in servers:
        server.stop(None)


def test_подсказка_доезжает_целиком(client):
    recognizer = FakeRecognizer(
        Recognition(
            name="Пуховик",
            category="outerwear",
            main_color="navy",
            extra_colors=["gray"],
            seasons=["winter"],
            warmth_level=5,
            waterproof=True,
        )
    )
    stub = client(recognizer)

    response = stub.RecognizeItem(vision_pb2.RecognizeItemRequest(photo=b"jpeg", content_type="image/jpeg"))

    assert recognizer.seen == b"jpeg"
    assert response.name == "Пуховик"
    assert response.category == "outerwear"
    assert response.main_color == "navy"
    assert list(response.extra_colors) == ["gray"]
    assert list(response.seasons) == ["winter"]
    assert response.warmth_level == 5
    assert response.waterproof is True


def test_ничего_не_узнали_тоже_ответ(client):
    stub = client(FakeRecognizer(Recognition()))

    response = stub.RecognizeItem(vision_pb2.RecognizeItemRequest(photo=b"jpeg", content_type="image/jpeg"))

    assert response.category == ""
    assert response.warmth_level == 0


def test_пустая_фотография_это_ошибка_запроса(client):
    stub = client(FakeRecognizer(Recognition()))

    with pytest.raises(grpc.RpcError) as failure:
        stub.RecognizeItem(vision_pb2.RecognizeItemRequest(photo=b"", content_type="image/jpeg"))

    assert failure.value.code() == grpc.StatusCode.INVALID_ARGUMENT


def test_поломка_модели_не_выдаёт_подробностей(client):
    stub = client(FakeRecognizer(RuntimeError("модель не загрузилась")))

    with pytest.raises(grpc.RpcError) as failure:
        stub.RecognizeItem(vision_pb2.RecognizeItemRequest(photo=b"jpeg", content_type="image/jpeg"))

    assert failure.value.code() == grpc.StatusCode.INTERNAL
    assert "модель не загрузилась" not in failure.value.details()
