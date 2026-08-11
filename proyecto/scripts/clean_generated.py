"""Elimina unicamente artefactos reproducibles generados por este proyecto."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
# La lista explicita evita borrar accidentalmente archivos que no genera el proyecto.
TARGETS = (
    ROOT / "data" / "experiment_results.csv",
    ROOT / "data" / "experiment_summary.json",
    ROOT / "figures" / "overhead.png",
    ROOT / "figures" / "recovery_rate.png",
    ROOT / "figures" / "outcomes.png",
    ROOT / "go-side" / "router.exe",
    ROOT / "go-side" / "router",
    *ROOT.glob("*_tabla_enrutamiento.csv"),
    *ROOT.glob("python-side/*_tabla_enrutamiento.csv"),
)

for target in TARGETS:
    if target.is_file():
        target.unlink()
        print(f"Eliminado: {target.relative_to(ROOT)}")
