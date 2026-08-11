# Router: Protocolo Link State

Cada nodo router implementa el plano de control (HELLO, LSA, flooding,
Dijkstra) y el plano de datos (Hamming(7,4) + forwarding) descritos en
[`docs/protocol.md`](docs/protocol.md). Un router corre dos hilos
principales en paralelo (routing y forwarding), mas un hilo de escucha que
despacha cada mensaje entrante segun su `"type"`.

## Componentes

- `python-side/router/` y `go-side/internal/router/` implementan el mismo
  nodo en Python y Go respectivamente (mismo protocolo, interoperables entre
  si por JSON).
- `legacy` de Lab 2 fue removido: no aplica al dominio de enrutamiento.

## Ejecutar un nodo

Sin `config.json`, el nodo pide los datos por consola (igual que en el
ejemplo del profesor: nombre, IP, puerto, vecinos, host adjunto):

```
make run-router-py NAME=U
make run-router-go NAME=U
```

Con un `config.json` ya escrito (ver formato en `docs/protocol.md`):

```
make run-router-py CONFIG=U_config.json
make run-router-go CONFIG=U_config.json
```

Al converger (30s tras arrancar), cada nodo escribe
`<nombre>_tabla_enrutamiento.csv` con su tabla de ruteo, y sigue
recalculandola cada vez que el grafo cambia (nuevo LSA, vecino caido).

### Cliente / servidor (hosts no-router)

Un host adjunto no corre el plano de control; solo usa a su router como
puerta de enlace:

```python
from router import host
host.send_via_gateway(self_id="100.w.w.w:6000", gateway_ip="100.x.x.x",
                       gateway_port=5000, to_id="100.v.v.v:6001", text="hola")
```

```python
host.run_server("100.v.v.v", 6001, on_message=print)
```

(Equivalente en Go: `router.SendViaGateway(...)` / `router.RunServer(...)`.)

## Testing y experimentos

```
make test              # test-python + test-go + test-integration
make test-python        # algoritmos y plano de datos en Python
make test-go             # idem en Go
make test-integration    # interoperabilidad Python <-> Go
make experiments         # overhead y robustez de Hamming(7,4) -> data/ + figures/
make clean               # borra artefactos generados (tablas de ruteo, figuras, binarios)
```

## Pendiente

- Confirmar con el profesor si la topologia se completa con 2 parejas (4
  personas, cada una operando 2 nodos router) o si se une a otra pareja —
  el enunciado exige topologias de 3 parejas / 6 integrantes.
- Fijar la asignacion final de puertos una vez conocido el numero de nodos.
- Acordar con las demas parejas de la topologia el formato exacto de
  `docs/protocol.md` (especialmente el envoltorio DATA con Hamming sobre el
  frame completo) antes de las pruebas de interoperabilidad.
