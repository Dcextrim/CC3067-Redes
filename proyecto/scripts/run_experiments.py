"""Experimento reproducible: robustez y overhead de Hamming(7,4) sobre
frames DATA representativos (ver docs/protocol.md, seccion 3.C).

Reemplaza el experimento de Lab 2 (que comparaba Hamming vs CRC-32 sobre
mensajes de ATM): Lab 3 solo usa Hamming, y lo aplica al frame {from,to,msg}
completo en vez de a un mensaje de aplicacion suelto.
"""

from __future__ import annotations

import csv
import json
from pathlib import Path
import random
import sys

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
from matplotlib.ticker import PercentFormatter  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router import codec  # noqa: E402
from router.algorithms import hamming, noise  # noqa: E402

# Tamanos representativos de frame DATA completo (json.dumps({from,to,msg})).
FRAME_SIZES_BYTES = (32, 64, 128, 256, 512)
ERROR_RATES = (0.0, 0.001, 0.005, 0.01, 0.02)
TRIALS = 300
SEED = 3067

DATA_PATH = ROOT / "data" / "experiment_results.csv"
SUMMARY_PATH = ROOT / "data" / "experiment_summary.json"
FIGURES = ROOT / "figures"


def frame_for_size(size: int) -> str:
    """JSON {from,to,msg} determinista cuyo tamano serializado se acerca a size bytes."""
    padding = "".join(chr(ord("A") + i % 26) for i in range(max(size - 40, 0)))
    return json.dumps({"from": "100.0.0.1:5000", "to": "100.0.0.2:5000", "msg": padding})


def run_experiment() -> list[dict[str, int | float | str]]:
    rng = random.Random(SEED)
    rows: list[dict[str, int | float | str]] = []
    for size in FRAME_SIZES_BYTES:
        message_bits = codec.codificar_mensaje(frame_for_size(size))
        original_frame = hamming.encode(message_bits)
        redundancy = len(original_frame) - len(message_bits)
        for error_rate in ERROR_RATES:
            exact = rejected = silent = corrected = total_flips = 0
            for _ in range(TRIALS):
                noisy_frame, flips = noise.aplicar_ruido(original_frame, error_rate, rng)
                total_flips += flips
                result = hamming.decode(noisy_frame, len(message_bits))
                if not result.valid:
                    rejected += 1
                elif result.data_bits == message_bits:
                    exact += 1
                else:
                    silent += 1
                if result.corrected:
                    corrected += 1
            rows.append(
                {
                    "frame_bytes": size,
                    "message_bits": len(message_bits),
                    "error_rate": error_rate,
                    "trials": TRIALS,
                    "encoded_bits": len(original_frame),
                    "redundancy_bits": redundancy,
                    "overhead_pct": 100.0 * redundancy / len(message_bits),
                    "mean_flips": total_flips / TRIALS,
                    "exact_recovery_rate": exact / TRIALS,
                    "rejected_rate": rejected / TRIALS,
                    "silent_corruption_rate": silent / TRIALS,
                    "corrected_rate": corrected / TRIALS,
                }
            )
    return rows


def save_data(rows: list[dict[str, int | float | str]]) -> None:
    DATA_PATH.parent.mkdir(parents=True, exist_ok=True)
    with DATA_PATH.open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(stream, fieldnames=list(rows[0]), lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)

    summary = {
        "seed": SEED,
        "trials_per_combination": TRIALS,
        "combinations": len(rows),
        "total_transmissions": len(rows) * TRIALS,
        "frame_sizes_bytes": FRAME_SIZES_BYTES,
        "error_rates": ERROR_RATES,
        "algorithm": "hamming",
    }
    SUMMARY_PATH.write_text(json.dumps(summary, indent=2), encoding="utf-8")


def configure_plot() -> None:
    plt.rcParams.update(
        {
            "font.family": "DejaVu Sans",
            "font.size": 9,
            "axes.titlesize": 11,
            "axes.labelsize": 9,
            "axes.spines.top": False,
            "axes.spines.right": False,
            "axes.grid": True,
            "axes.grid.axis": "y",
            "grid.alpha": 0.22,
            "legend.frameon": False,
        }
    )


def plot_overhead(rows: list[dict[str, int | float | str]]) -> None:
    """Redundancia relativa de Hamming(7,4) frente al tamano del frame."""
    unique = [row for row in rows if float(row["error_rate"]) == 0.0]
    fig, axis = plt.subplots(figsize=(7.2, 4.0))
    x = [int(row["frame_bytes"]) for row in unique]
    y = [float(row["overhead_pct"]) for row in unique]
    axis.plot(x, y, marker="o", linewidth=2, color="#007C83", label="Hamming(7,4)")
    for x_value, y_value in zip(x, y):
        axis.annotate(f"{y_value:.1f}%", (x_value, y_value), xytext=(0, 6),
                       textcoords="offset points", ha="center", fontsize=7)
    axis.set_xscale("log", base=2)
    axis.set_xticks(FRAME_SIZES_BYTES, [str(v) for v in FRAME_SIZES_BYTES])
    axis.set_xlabel("Tamaño del frame DATA {from,to,msg} (bytes, escala log2)")
    axis.set_ylabel("Redundancia / datos originales (%)")
    axis.set_title("Overhead de Hamming(7,4) según el tamaño del frame")
    axis.legend()
    fig.tight_layout()
    fig.savefig(FIGURES / "overhead.png", dpi=200, bbox_inches="tight")
    plt.close(fig)


def plot_recovery(rows: list[dict[str, int | float | str]]) -> None:
    """Recuperacion exacta por tamano de frame y tasa de error."""
    palette = plt.get_cmap("viridis")
    fig, axis = plt.subplots(figsize=(7.2, 4.0))
    for color_index, size in enumerate(FRAME_SIZES_BYTES):
        selected = [row for row in rows if int(row["frame_bytes"]) == size]
        x = list(range(len(selected)))
        y = [float(row["exact_recovery_rate"]) for row in selected]
        axis.plot(x, y, marker="o", linewidth=1.7,
                  color=palette(color_index / (len(FRAME_SIZES_BYTES) - 1)), label=f"{size} B")
    axis.set_xlabel("Probabilidad de flip por bit (%)")
    axis.set_xticks(range(len(ERROR_RATES)), ["0", "0.1", "0.5", "1", "2"])
    axis.set_ylim(-0.03, 1.03)
    axis.yaxis.set_major_formatter(PercentFormatter(1.0))
    axis.set_ylabel("Frames DATA recuperados exactamente")
    axis.legend(title="Frame", ncol=1, loc="upper right")
    axis.set_title("Recuperación exacta de Hamming(7,4) por tamaño de frame y ruido")
    fig.tight_layout()
    fig.savefig(FIGURES / "recovery_rate.png", dpi=200, bbox_inches="tight")
    plt.close(fig)


def plot_outcomes(rows: list[dict[str, int | float | str]]) -> None:
    """Entregas exactas, rechazos y corrupcion silenciosa a 128 bytes."""
    size = 128
    selected = [row for row in rows if int(row["frame_bytes"]) == size]
    labels = [f"{float(row['error_rate']) * 100:g}%" for row in selected]
    exact_values = [float(row["exact_recovery_rate"]) for row in selected]
    rejected_values = [float(row["rejected_rate"]) for row in selected]
    silent_values = [float(row["silent_corruption_rate"]) for row in selected]

    positions = list(range(len(labels)))
    fig, axis = plt.subplots(figsize=(7.2, 4.0))
    axis.bar(positions, exact_values, color="#2A9D8F", label="Recuperado exacto")
    axis.bar(positions, rejected_values, bottom=exact_values, color="#8D99AE", label="Rechazado")
    combined = [a + b for a, b in zip(exact_values, rejected_values)]
    axis.bar(positions, silent_values, bottom=combined, color="#C44536", label="Corrupción silenciosa")
    axis.set_xticks(positions, labels)
    axis.set_xlabel("Probabilidad de flip por bit")
    axis.set_ylabel("Proporción de transmisiones")
    axis.yaxis.set_major_formatter(PercentFormatter(1.0))
    axis.set_title(f"Resultado de {TRIALS} envíos por condición (frame de {size} bytes)")
    axis.legend(ncol=3, loc="upper center", bbox_to_anchor=(0.5, -0.18))
    axis.set_ylim(0, 1)
    fig.tight_layout()
    fig.savefig(FIGURES / "outcomes.png", dpi=200, bbox_inches="tight")
    plt.close(fig)


def main() -> None:
    FIGURES.mkdir(parents=True, exist_ok=True)
    configure_plot()
    rows = run_experiment()
    save_data(rows)
    plot_overhead(rows)
    plot_recovery(rows)
    plot_outcomes(rows)
    print(f"Generadas {len(rows)} combinaciones y {len(rows) * TRIALS} transmisiones simuladas.")
    print(f"Datos: {DATA_PATH.relative_to(ROOT)}")
    print(f"Graficas: {FIGURES.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
