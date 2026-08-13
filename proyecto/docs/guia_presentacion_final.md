# Guía para la presentación final (con la otra pareja)

## 0. Antes del día de la presentación

- **Confirmen con Javier/Dilary quién corre qué nodo** en la topología de 6 (recordatorio: propuesta previa era 5000-5002 ustedes, 5003-5005 ellos, 5006-5007 nodo(s) simulado(s) del 3er "pareja" faltante — confirmen que sigue vigente).
- **Prueben Tailscale de nuevo el mismo día**, no confíen en la prueba anterior. Cada quien corre `tailscale ip -4` y comparten las IPs actuales.
- **Decidan dónde conectan el ATM y el Banco**: lo ideal es que uno cuelgue de un router suyo y el otro de un router de la otra pareja, para que el datagrama cruce Tailscale y varios saltos reales — es el escenario más convincente para el profesor, y es justo lo que pide "todos los nodos involucrados deben mostrar que por ellos pasan los datagramas".
- Tengan **`docs/protocol.md`** compartido con la otra pareja, especialmente la sección 3.D que documenta el formato del ATM/Banco — aunque ese formato es interno de su pareja, si el ATM/Banco de ustedes cruza por un router de ellos, ellos solo necesitan reenviar, no entender `msg`.
- Repasen que **`go build ./... && go vet ./... && go test ./...`** esté limpio justo antes de salir de casa.

## 1. Cuando el profesor asigne la topología (el día del examen)

El profesor da una topología de 6 nodos con costos. Tienen que traducirla a JSON rápido:

1. Cada quien edita **su propio** `configs/demo/<NODO>.json` (esa carpeta no se versiona, es la pensada para esto — ver `configs/examples/A.tailscale.example.json` como plantilla).
2. Reemplacen `IP_LOCAL_TAILSCALE` por su propia IP de Tailscale, y `IP_REMOTA_TAILSCALE` por la del vecino correspondiente (puede ser un nodo de la otra pareja).
3. Respeten los **costos exactos** que dio el profesor en los `neighbors[].cost` — Dijkstra depende de eso para que la ruta más corta coincida con lo que él espera ver.
4. Si a alguno de ustedes le toca el nodo con el ATM o el Banco, agreguen el `attached_host` con `role: "atm"` o `role: "bank"` y el puerto que decidan (ej. 6000/6001, o algo en el rango 6000+ para no chocar con los routers).

## 2. Levantar el sistema

Orden recomendado (cada quien corre lo suyo, en paralelo con la otra pareja):

```bash
# cada persona, en su propia máquina, para su(s) nodo(s):
go run ./cmd/router -config configs/demo/<NODO>.json -noise 0
```

- Usen `-noise 0` para la corrida "oficial" — sin ruido, comportamiento determinista, es la versión ya probada. Si el profesor pide ver el manejo de ruido, es un flag aparte (`-noise 1/1000`), no lo mezclen con la demo principal.
- **Esperen la convergencia real (~35-40s)** antes de mostrar nada: cada router va a imprimir `[ROUTER X] Converged. Recomputando rutas ante cada cambio de topologia.` — ese es el semáforo verde.
- Quien tenga el Banco:
  ```bash
  go run ./cmd/bank -ip <IP_TAILSCALE_PROPIA> -port 6001 -gateway-ip <IP_ROUTER_GATEWAY> -gateway-port <PUERTO_ROUTER> -noise 0
  ```
- Quien tenga el ATM (dejarlo para el final, es lo que se corre en vivo frente al profesor):
  ```bash
  go run ./cmd/atm -ip <IP_TAILSCALE_PROPIA> -port 6000 -gateway-ip <IP_ROUTER_GATEWAY> -gateway-port <PUERTO_ROUTER> -bank <IP_TAILSCALE_BANCO>:6001
  ```

## 3. Qué mostrar, mapeado a los criterios de calificación

**a) "Ejecutar el plano de control y verificar LSA, flooding, tablas de ruteo"**
- Dejen las terminales de los 6 routers visibles (o compartan pantalla rotando). Señalen en vivo:
  - Líneas `[ROUTER X] LSA: ...` (construcción del LSA propio)
  - Líneas `[NETWORK X] LSA from ... stored -> flooding onward` (flooding real, no solo local)
  - Líneas `[ROUTER X] Shortest paths (Dijkstra): ...` con costo y next-hop por destino
  - Abran el CSV resultante: `cat <NODO>_tabla_enrutamiento.csv` — es literal lo que pide el enunciado.

**b) "Ejecutar el plano de datos y verificar que se usa la ruta más corta"**
- Antes de correr el ATM, calculen a mano (o lean del log de Dijkstra) cuál debería ser la ruta óptima entre el nodo del ATM y el del Banco, y díganla en voz alta antes de correrlo — así el profesor ve que ustedes también saben cuál es la ruta esperada, no solo el sistema.
- Corran el flujo completo del cajero (login → retiro → logout) en vivo.

**c) "Todos los nodos involucrados deben mostrar que por ellos pasan los datagramas"**
- Cada router intermedio imprime `[NETWORK X] DATA reenviada -> siguiente salto IP:PUERTO`. Tengan las terminales de **todos** los routers de la ruta visibles al mismo tiempo (o graben pantalla si son muchas ventanas) para que se vea en simultáneo que el datagrama fue pasando nodo por nodo, no solo que llegó al final.
- El Banco también imprime `[BANCO] atmID -> comando | respuesta: ...` — sirve como confirmación adicional de entrega end-to-end.

## 4. Plan B si algo no conecta

- El enunciado da **una segunda oportunidad** antes de aplazar con penalización — no entren en pánico si falla el primer intento.
- Diagnóstico rápido en orden:
  1. `tailscale status` — ¿todos aparecen conectados?
  2. ¿Coinciden IP y puerto en el JSON de cada quien con lo que el vecino tiene configurado como *su* IP/puerto? (typo común: invertir el config file de alguien)
  3. ¿Ya pasaron los ~35-40s de convergencia antes de correr el ATM?
  4. Revisen que nadie tenga un firewall bloqueando el puerto localmente (`sudo ufw status` en Linux, Firewall de Windows si alguien usa Windows).
- Si el ATM/Banco cruza con la otra pareja y falla, tengan como respaldo poder demostrar el flujo **solo dentro de su propia sub-topología** (sus 2-3 routers) para no perder toda la demo por un problema de la otra pareja — pregúntenle al profesor si eso cuenta parcialmente, o simplemente muestren ambas cosas si da tiempo.

## 5. Checklist de 5 minutos antes de arrancar

- [ ] `go build ./...` sin errores en las máquinas de ambas parejas
- [ ] Tailscale conectado, IPs confirmadas entre todos
- [ ] Configs de `configs/demo/` con la topología real que dio el profesor (no la de prueba A-F)
- [ ] Los 6 routers corriendo y convergidos (`Converged` visto en cada log)
- [ ] Banco escuchando, mostrando `[BANCO] Escuchando en ...`
- [ ] Terminales de los routers de la ruta ATM→Banco visibles/organizadas para mostrar el hop-by-hop
- [ ] Saben de memoria cuál es la ruta óptima esperada, para poder contrastarla con lo que muestre el sistema
