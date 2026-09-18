"""Считает эмбеддинги подписей при сборке образа: в рантайме текстовая часть модели не нужна."""

import sys

import numpy as np
import onnxruntime
from tokenizers import Tokenizer

from app.recognize.labels import PROMPTS

CONTEXT_LENGTH = 77


def main(model_path: str, tokenizer_path: str, out_path: str) -> None:
    tokenizer = Tokenizer.from_file(tokenizer_path)
    tokenizer.enable_truncation(max_length=CONTEXT_LENGTH)
    tokenizer.enable_padding(length=CONTEXT_LENGTH)
    ids = np.array([encoded.ids for encoded in tokenizer.encode_batch(list(PROMPTS))], dtype=np.int64)

    session = onnxruntime.InferenceSession(model_path, providers=["CPUExecutionProvider"])
    (embeds,) = session.run(None, {"input_ids": ids})
    np.save(out_path, (embeds / np.linalg.norm(embeds, axis=-1, keepdims=True)).astype(np.float32))


if __name__ == "__main__":
    main(*sys.argv[1:4])
