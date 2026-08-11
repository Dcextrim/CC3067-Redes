package router

import (
	"fmt"
	"sync"
	"testing"

	"cc3067/lab3/go-side/internal/router/control"
)

// TestConcurrentActiveNeighborsAndRoutes ejercita en paralelo los mismos
// caminos que en produccion corren en goroutines distintas (onHello desde
// routingLoop, buildAndFloodOwnLSA desde el timer del primer HELLO, y el
// intercambio de routes entre convergenceTimer y forwardingLoop). Sin el
// mutex que protege activeNeighbors/routes, `go test -race` falla aqui.
func TestConcurrentActiveNeighborsAndRoutes(t *testing.T) {
	config := NodeConfig{
		Name: "T",
		IP:   "127.0.0.1",
		Port: 5900,
		Neighbors: []Neighbor{
			{IP: "127.0.0.1", Port: 65501, Cost: 1},
			{IP: "127.0.0.1", Port: 65502, Cost: 1},
		},
	}
	n := NewNode(config, t.TempDir()+"/routing_table.csv")

	var wg sync.WaitGroup

	// En produccion, onHello solo lo llama la goroutine unica de routingLoop
	// (un for-range secuencial sobre el canal) -- nunca corre concurrente
	// consigo misma, asi que aqui tambien se simula desde una sola goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			n.onHello(fmt.Sprintf("127.0.0.1:%d", 65501+i%2))
		}
	}()

	// buildAndFloodOwnLSA corre en su propia goroutine (time.AfterFunc),
	// concurrente con onHello -- ambas tocan activeNeighbors.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.buildAndFloodOwnLSA()
		}()
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n.mu.Lock()
			n.routes = map[string]control.Route{"X": {Destination: "X", Cost: i}}
			n.mu.Unlock()
		}(i)
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.mu.RLock()
			_ = n.routes
			n.mu.RUnlock()
		}()
	}
	wg.Wait()
}
