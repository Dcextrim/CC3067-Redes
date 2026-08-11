package router

import (
	"testing"
	"time"

	"cc3067/lab3/go-side/internal/router/control"
)

// TestMultiRouterConvergence levanta 3 nodos router reales (sockets TCP de
// verdad en 127.0.0.1, sin mocks) en topologia de linea A-B-C, deja que
// intercambien HELLO/LSA por la red y verifica que la tabla de ruteo de cada
// nodo converge a las rutas mas cortas correctas (Dijkstra de punta a punta,
// no solo el algoritmo en aislamiento). Usa temporizadores acelerados (los de
// produccion, 10s/30s, harian la prueba demasiado lenta).
func TestMultiRouterConvergence(t *testing.T) {
	dir := t.TempDir()

	configA := NodeConfig{
		Name: "A", IP: "127.0.0.1", Port: 15910,
		Neighbors: []Neighbor{{IP: "127.0.0.1", Port: 15911, Cost: 2}},
	}
	configB := NodeConfig{
		Name: "B", IP: "127.0.0.1", Port: 15911,
		Neighbors: []Neighbor{
			{IP: "127.0.0.1", Port: 15910, Cost: 2},
			{IP: "127.0.0.1", Port: 15912, Cost: 3},
		},
	}
	configC := NodeConfig{
		Name: "C", IP: "127.0.0.1", Port: 15912,
		Neighbors: []Neighbor{{IP: "127.0.0.1", Port: 15911, Cost: 3}},
	}

	accelerate := func(n *Node) {
		n.HelloInterval = 100 * time.Millisecond
		n.LSADelayAfterFirstHello = 200 * time.Millisecond
		n.ConvergenceWait = 800 * time.Millisecond
		n.NeighborCheckInterval = 150 * time.Millisecond
		n.NeighborTimeout = 1 * time.Second
		n.RouteRecomputeInterval = 300 * time.Millisecond
		n.LSARebuildDebounce = 100 * time.Millisecond
	}

	nodeA := NewNode(configA, dir+"/A_tabla_enrutamiento.csv")
	nodeB := NewNode(configB, dir+"/B_tabla_enrutamiento.csv")
	nodeC := NewNode(configC, dir+"/C_tabla_enrutamiento.csv")
	accelerate(nodeA)
	accelerate(nodeB)
	accelerate(nodeC)

	nodeA.Start()
	nodeB.Start()
	nodeC.Start()
	t.Cleanup(nodeA.Stop)
	t.Cleanup(nodeB.Stop)
	t.Cleanup(nodeC.Stop)

	idA, idB, idC := configA.ID(), configB.ID(), configC.ID()

	waitForRoute := func(t *testing.T, n *Node, destination string) control.Route {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			n.mu.RLock()
			route, ok := n.routes[destination]
			n.mu.RUnlock()
			if ok {
				return route
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("timeout esperando ruta hacia %s", destination)
		return control.Route{}
	}

	routeAtoC := waitForRoute(t, nodeA, idC)
	if routeAtoC.Cost != 5 {
		t.Errorf("A->C: costo esperado 5 (via B), obtuvo %d", routeAtoC.Cost)
	}
	if routeAtoC.NextHopPort != configB.Port {
		t.Errorf("A->C: siguiente salto esperado B (%d), obtuvo puerto %d", configB.Port, routeAtoC.NextHopPort)
	}

	routeCtoA := waitForRoute(t, nodeC, idA)
	if routeCtoA.Cost != 5 {
		t.Errorf("C->A: costo esperado 5 (via B), obtuvo %d", routeCtoA.Cost)
	}
	if routeCtoA.NextHopPort != configB.Port {
		t.Errorf("C->A: siguiente salto esperado B (%d), obtuvo puerto %d", configB.Port, routeCtoA.NextHopPort)
	}

	routeAtoB := waitForRoute(t, nodeA, idB)
	if routeAtoB.Cost != 2 {
		t.Errorf("A->B: costo esperado 2, obtuvo %d", routeAtoB.Cost)
	}

	routeBtoC := waitForRoute(t, nodeB, idC)
	if routeBtoC.Cost != 3 {
		t.Errorf("B->C: costo esperado 3, obtuvo %d", routeBtoC.Cost)
	}
}
