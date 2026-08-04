"""Simulador de canal binario simetrico."""

import random


def parse_probability(value: str) -> float:
    text = value.strip()
    try:
        if "/" in text:
            numerator_text, denominator_text = text.split("/", 1)
            denominator = float(denominator_text)
            if denominator == 0:
                raise ValueError
            probability = float(numerator_text) / denominator
        else:
            probability = float(text)
    except ValueError as exc:
        raise ValueError("use una probabilidad decimal o una fraccion como 1/100") from exc
    if not 0.0 <= probability <= 1.0:
        raise ValueError("la probabilidad debe estar entre 0 y 1")
    return probability


def aplicar_ruido(bits: str, probability: float, rng: random.Random | None = None) -> tuple[str, int]:
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")
    if not 0.0 <= probability <= 1.0:
        raise ValueError("la probabilidad debe estar entre 0 y 1")
    source = rng or random
    output: list[str] = []
    flips = 0
    for bit in bits:
        if source.random() < probability:
            output.append("1" if bit == "0" else "0")
            flips += 1
        else:
            output.append(bit)
    return "".join(output), flips
