"""Elimina unicamente artefactos reproducibles generados por este proyecto."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TARGETS = (
    ROOT / "data" / "experiment_results.csv",
    ROOT / "data" / "experiment_summary.json",
    ROOT / "figures" / "overhead.png",
    ROOT / "figures" / "recovery_rate.png",
    ROOT / "figures" / "outcomes.png",
    ROOT / "go-side" / "bank-server.exe",
    ROOT / "go-side" / "bank-server",
)

for target in TARGETS:
    if target.is_file():
        target.unlink()
        print(f"Eliminado: {target.relative_to(ROOT)}")
