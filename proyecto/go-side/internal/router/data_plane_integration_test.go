package router

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cc3067/lab3/go-side/internal/router/control"
)

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func accelerateNode(n *Node) {
	n.HelloInterval = 100 * time.Millisecond
	n.LSADelayAfterFirstHello = 200 * time.Millisecond
	n.ConvergenceWait = 800 * time.Millisecond
	n.NeighborCheckInterval = 150 * time.Millisecond
	n.NeighborTimeout = time.Second
	n.RouteRecomputeInterval = 300 * time.Millisecond
	n.LSARebuildDebounce = 100 * time.Millisecond
}

func waitForDestination(t *testing.T, n *Node, destination string) control.Route {
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
	t.Fatalf("timeout esperando ruta hacia %s en %s", destination, n.Config.Name)
	return control.Route{}
}

// TestClientThroughRoutersToServer cubre el flujo completo exigido por la
// guia: cliente no-router -> gateway A -> B -> C -> servidor no-router. Los
// tres routers intercambian HELLO/LSA por sockets reales, escriben sus CSV y
// el forwarding consulta esos archivos antes de reenviar la trama Hamming.
func TestClientThroughRoutersToServer(t *testing.T) {
	ports := []int{freeTCPPort(t), freeTCPPort(t), freeTCPPort(t)}
	serverListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverListener.Close() })
	serverPort := serverListener.Addr().(*net.TCPAddr).Port
	clientPort := freeTCPPort(t)

	received := make(chan control.Payload, 1)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- ServeHost(serverListener, func(payload control.Payload) { received <- payload })
	}()

	clientHost := &AttachedHost{Role: "client", IP: "127.0.0.1", Port: clientPort, Cost: 1}
	serverHost := &AttachedHost{Role: "server", IP: "127.0.0.1", Port: serverPort, Cost: 1}
	configA := NodeConfig{
		Name: "A", IP: "127.0.0.1", Port: ports[0], AttachedHost: clientHost,
		Neighbors: []Neighbor{{IP: "127.0.0.1", Port: ports[1], Cost: 2}},
	}
	configB := NodeConfig{
		Name: "B", IP: "127.0.0.1", Port: ports[1],
		Neighbors: []Neighbor{
			{IP: "127.0.0.1", Port: ports[0], Cost: 2},
			{IP: "127.0.0.1", Port: ports[2], Cost: 3},
		},
	}
	configC := NodeConfig{
		Name: "C", IP: "127.0.0.1", Port: ports[2], AttachedHost: serverHost,
		Neighbors: []Neighbor{{IP: "127.0.0.1", Port: ports[1], Cost: 3}},
	}

	directory := t.TempDir()
	nodes := []*Node{
		NewNode(configA, filepath.Join(directory, "A_tabla_enrutamiento.csv")),
		NewNode(configB, filepath.Join(directory, "B_tabla_enrutamiento.csv")),
		NewNode(configC, filepath.Join(directory, "C_tabla_enrutamiento.csv")),
	}
	for _, node := range nodes {
		accelerateNode(node)
		node.Start()
		t.Cleanup(node.Stop)
	}

	route := waitForDestination(t, nodes[0], serverHost.ID())
	if route.Cost != 6 || route.NextHopPort != configB.Port {
		t.Fatalf("ruta A->servidor inesperada: %+v", route)
	}
	for _, node := range nodes {
		if _, err := os.Stat(node.CSVPath); err != nil {
			t.Fatalf("%s no genero su tabla CSV: %v", node.Config.Name, err)
		}
	}

	want := control.Payload{From: clientHost.ID(), To: serverHost.ID(), Msg: "mensaje end-to-end"}
	if err := SendViaGateway(want.From, configA.IP, configA.Port, want.To, want.Msg); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-received:
		if got != want {
			t.Fatalf("payload recibido = %+v; se esperaba %+v", got, want)
		}
	case err := <-serverErrors:
		t.Fatalf("el servidor termino antes de recibir el mensaje: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal(fmt.Sprintf("timeout esperando DATA en %s", serverHost.ID()))
	}
}
