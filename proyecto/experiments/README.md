# Experimento reproducible de ruido y Hamming

Esta carpeta contiene resultados generados exclusivamente con la
implementacion Go usada por los routers. La simulacion codifica frames DATA
completos con Hamming(7,4), voltea cada bit de forma independiente con la
probabilidad configurada y comprueba si el frame original se recupera.

Para regenerar todos los archivos desde `proyecto/`:

```powershell
go run ./cmd/experiment
```

La semilla pseudoaleatoria y las condiciones estan fijadas en
`internal/experiment`, por lo que dos ejecuciones producen los mismos datos:

- `data/experiment_results.csv`: una fila por tamano real de frame y tasa de
  error.
- `data/experiment_summary.json`: semilla, condiciones y modelo del canal.
- `figures/*.svg`: overhead, recuperacion exacta y resultado de las
  transmisiones. Son SVG generados con la biblioteca estandar de Go.

`exact_recovery_rate` es la medida principal. `silent_corruption_rate` recuerda
la limitacion del codigo: Hamming(7,4) corrige un error por bloque de siete
bits, pero varios flips en el mismo bloque pueden producir datos incorrectos
sin que el codigo los distinga.
