"""Ejecuta experimentos deterministas y genera CSV, JSON y graficas."""

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

from atm.layers import link, noise, presentation  # noqa: E402

MESSAGE_SIZES = (1, 4, 16, 64, 256)
ERROR_RATES = (0.0, 0.001, 0.005, 0.01, 0.02)
ALGORITHMS = ("hamming", "crc32")
TRIALS = 300
SEED = 3067

DATA_PATH = ROOT / "data" / "experiment_results.csv"
SUMMARY_PATH = ROOT / "data" / "experiment_summary.json"
FIGURES = ROOT / "figures"


def message_for_size(size: int) -> str:
    return "".join(chr(ord("A") + index % 26) for index in range(size))


def run_experiment() -> list[dict[str, int | float | str]]:
    rng = random.Random(SEED)
    rows: list[dict[str, int | float | str]] = []
    for algorithm in ALGORITHMS:
        for size in MESSAGE_SIZES:
            message_bits = presentation.codificar_mensaje(message_for_size(size))
            original_frame = link.calcular_integridad(message_bits, algorithm)
            redundancy = len(original_frame) - len(message_bits)
            for error_rate in ERROR_RATES:
                exact = rejected = silent = corrected = total_flips = 0
                for _ in range(TRIALS):
                    noisy_frame, flips = noise.aplicar_ruido(original_frame, error_rate, rng)
                    total_flips += flips
                    result = link.verificar_integridad(
                        noisy_frame, algorithm, len(message_bits)
                    )
                    if not result.ok:
                        rejected += 1
                    elif result.message_bits == message_bits:
                        exact += 1
                    else:
                        silent += 1
                    if result.corrected:
                        corrected += 1

                rows.append(
                    {
                        "algorithm": algorithm,
                        "message_bytes": size,
                        "message_bits": len(message_bits),
                        "error_rate": error_rate,
                        "trials": TRIALS,
                        "frame_bits": len(original_frame),
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
        writer = csv.DictWriter(stream, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)

    summary = {
        "seed": SEED,
        "trials_per_combination": TRIALS,
        "combinations": len(rows),
        "total_transmissions": len(rows) * TRIALS,
        "message_sizes_bytes": MESSAGE_SIZES,
        "error_rates": ERROR_RATES,
        "algorithms": ALGORITHMS,
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
    unique = [row for row in rows if float(row["error_rate"]) == 0.0]
    colors = {"hamming": "#007C83", "crc32": "#D2691E"}
    fig, axis = plt.subplots(figsize=(7.2, 4.0))
    for algorithm in ALGORITHMS:
        selected = [row for row in unique if row["algorithm"] == algorithm]
        x = [int(row["message_bytes"]) for row in selected]
        y = [float(row["overhead_pct"]) for row in selected]
        label = "Hamming" if algorithm == "hamming" else "CRC-32"
        axis.plot(x, y, marker="o", linewidth=2, color=colors[algorithm], label=label)
        label_offset = 5 if algorithm == "hamming" else 15
        for x_value, y_value in zip(x, y):
            axis.annotate(
                f"{y_value:.1f}%", (x_value, y_value), xytext=(0, label_offset),
                textcoords="offset points", ha="center", fontsize=7,
            )
    axis.set_xscale("log", base=2)
    axis.set_xticks(MESSAGE_SIZES, [str(value) for value in MESSAGE_SIZES])
    axis.set_xlabel("Tamaño del mensaje (bytes, escala log2)")
    axis.set_ylabel("Redundancia / datos originales (%)")
    axis.set_title("Overhead de integridad según el tamaño del mensaje")
    axis.legend()
    axis.set_ylim(0, 430)
    fig.tight_layout()
    fig.savefig(FIGURES / "overhead.png", dpi=200, bbox_inches="tight")
    plt.close(fig)


def plot_recovery(rows: list[dict[str, int | float | str]]) -> None:
    palette = plt.get_cmap("viridis")
    fig, axes = plt.subplots(1, 2, figsize=(8.2, 3.8), sharey=True)
    for axis, algorithm in zip(axes, ALGORITHMS):
        for color_index, size in enumerate(MESSAGE_SIZES):
            selected = [
                row for row in rows
                if row["algorithm"] == algorithm and int(row["message_bytes"]) == size
            ]
            x = list(range(len(selected)))
            y = [float(row["exact_recovery_rate"]) for row in selected]
            axis.plot(
                x, y, marker="o", linewidth=1.7,
                color=palette(color_index / (len(MESSAGE_SIZES) - 1)),
                label=f"{size} B",
            )
        axis.set_title("Hamming" if algorithm == "hamming" else "CRC-32")
        axis.set_xlabel("Probabilidad de flip por bit (%)")
        axis.set_xticks(range(len(ERROR_RATES)), ["0", "0.1", "0.5", "1", "2"])
        axis.set_ylim(-0.03, 1.03)
        axis.yaxis.set_major_formatter(PercentFormatter(1.0))
    axes[0].set_ylabel("Mensajes recuperados exactamente")
    axes[1].legend(title="Mensaje", ncol=1, loc="upper right")
    fig.suptitle("Recuperación exacta: algoritmo, longitud y ruido", y=1.02, fontsize=12)
    fig.tight_layout()
    fig.savefig(FIGURES / "recovery_rate.png", dpi=200, bbox_inches="tight")
    plt.close(fig)


def plot_outcomes(rows: list[dict[str, int | float | str]]) -> None:
    size = 16
    selected = [row for row in rows if int(row["message_bytes"]) == size]
    labels: list[str] = []
    exact_values: list[float] = []
    rejected_values: list[float] = []
    silent_values: list[float] = []
    for error_rate in ERROR_RATES:
        for algorithm in ALGORITHMS:
            row = next(
                item for item in selected
                if item["algorithm"] == algorithm and float(item["error_rate"]) == error_rate
            )
            short = "Ham" if algorithm == "hamming" else "CRC"
            labels.append(f"{short}\n{error_rate * 100:g}%")
            exact_values.append(float(row["exact_recovery_rate"]))
            rejected_values.append(float(row["rejected_rate"]))
            silent_values.append(float(row["silent_corruption_rate"]))

    positions = list(range(len(labels)))
    fig, axis = plt.subplots(figsize=(8.2, 4.2))
    axis.bar(positions, exact_values, color="#2A9D8F", label="Recuperado exacto")
    axis.bar(positions, rejected_values, bottom=exact_values, color="#8D99AE", label="Rechazado")
    combined = [a + b for a, b in zip(exact_values, rejected_values)]
    axis.bar(positions, silent_values, bottom=combined, color="#C44536", label="Corrupcion silenciosa")
    axis.set_xticks(positions, labels)
    axis.set_ylabel("Proporción de transmisiones")
    axis.yaxis.set_major_formatter(PercentFormatter(1.0))
    axis.set_title("Resultado de 300 envíos por condición (mensaje de 16 bytes)")
    axis.legend(ncol=3, loc="upper center", bbox_to_anchor=(0.5, -0.16))
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
