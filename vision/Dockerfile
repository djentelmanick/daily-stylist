# Стадия модели: веса берутся уже в ONNX, и здесь же один раз считаются эмбеддинги
# подписей. В итоговый образ из неё едет только энкодер картинок и готовые эмбеддинги.
FROM python:3.12-slim AS model

ARG MODEL_REPO=Marqo/marqo-fashionCLIP
ARG MODEL_REVISION=44f4c655124ed71e90cbf528d82f508d69d8a81b
ARG MODEL_URL=https://huggingface.co/${MODEL_REPO}/resolve/${MODEL_REVISION}

WORKDIR /build

COPY pyproject.toml ./
RUN mkdir -p app && touch app/__init__.py \
    && pip install --no-cache-dir '.[export]' \
    && rm -rf app

ADD ${MODEL_URL}/onnx/vision_model_int8.onnx /models/vision_model.onnx
ADD ${MODEL_URL}/onnx/text_model_int8.onnx text_model.onnx
ADD ${MODEL_URL}/tokenizer.json tokenizer.json

COPY app ./app
COPY tools ./tools
RUN python -m tools.embed_prompts text_model.onnx tokenizer.json /models/prompts.npy


FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY pyproject.toml ./
RUN mkdir -p app && touch app/__init__.py \
    && pip install --no-cache-dir . \
    && pip uninstall -y daily-stylist-vision \
    && rm -rf app

COPY --from=model /models /models
COPY healthcheck.py ./
COPY app ./app

CMD ["python", "-m", "app.main"]
