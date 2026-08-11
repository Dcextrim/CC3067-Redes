package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"cc3067/lab3/go-side/internal/router/algorithms"
	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

const (
	HelloIntervalS            = 10 * time.Second
	LSADelayAfterFirstHelloS  = 5 * time.Second
	ConvergenceWaitS          = 30 * time.Second
)

// Node es un router: goroutine de escucha + goroutine de routing + goroutine
// de forwarding corriendo en paralelo (ver seccion 3.3 del enunciado).
type Node struct {
	Config  NodeConfig
	CSVPath string

	store *control.LinkStateStore

	// mu protege activeNeighbors y routes: se leen/escriben desde goroutines
	// distintas (routingLoop, el timer de buildAndFloodOwnLSA, convergenceTimer
	// y forwardingLoop corren en paralelo).
	mu              sync.RWMutex
	activeNeighbors map[string]bool
	routes          map[string]control.Route

	routingQueue    chan map[string]interface{}
	forwardingQueue chan control.DataEnvelope
	firstHelloSeen  bool
}

// NewNode crea un nodo listo para arrancar con Start/RunForever.
func NewNode(config NodeConfig, csvPath string) *Node {
	if csvPath == "" {
		csvPath = fmt.Sprintf("%s_routing_table.csv", config.Name)
	}
	return &Node{
		Config:          config,
		CSVPath:         csvPath,
		store:           control.NewLinkStateStore(config.ID()),
		activeNeighbors: map[string]bool{},
		routes:          map[string]control.Route{},
		routingQueue:    make(chan map[string]interface{}, 64),
		forwardingQueue: make(chan control.DataEnvelope, 64),
	}
}

// Start lanza las goroutines del nodo y retorna de inmediato.
func (n *Node) Start() {
	go n.listenLoop()
	go n.helloLoop()
	go n.routingLoop()
	go n.forwardingLoop()
	go n.convergenceTimer()
}

// RunForever arranca el nodo y bloquea el proceso principal.
func (n *Node) RunForever() {
	n.Start()
	select {}
}

// SendMessage abre una conexion TCP corta, envia un mensaje JSON y cierra.
func (n *Node) SendMessage(ip string, port int, message interface{}) {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		log.Printf("[NETWORK %s] no se pudo enviar a %s (%v)", n.Config.ID(), address, err)
		return
	}
	defer conn.Close()
	raw, err := json.Marshal(message)
	if err != nil {
		log.Printf("[NETWORK %s] error serializando mensaje: %v", n.Config.ID(), err)
		return
	}
	conn.Write(append(raw, '\n'))
}

func (n *Node) listenLoop() {
	address := fmt.Sprintf("%s:%d", n.Config.IP, n.Config.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[NETWORK %s] no se pudo escuchar en %s: %v", n.Config.ID(), address, err)
	}
	log.Printf("[NETWORK %s] Listening", n.Config.ID())
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go n.handleConnection(conn)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && line == "" {
		return
	}
	var generic map[string]interface{}
	if err := json.Unmarshal([]byte(line), &generic); err != nil {
		log.Printf("[NETWORK %s] mensaje invalido descartado", n.Config.ID())
		return
	}
	if generic["type"] == control.DATA {
		var envelope control.DataEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err == nil {
			n.forwardingQueue <- envelope
		}
		return
	}
	n.routingQueue <- generic
}

// -- plano de control (goroutine de routing) --------------------------------

func (n *Node) helloLoop() {
	for {
		for _, neighbor := range n.Config.Neighbors {
			n.SendMessage(neighbor.IP, neighbor.Port, control.BuildHello(n.Config.ID()))
		}
		time.Sleep(HelloIntervalS)
	}
}

func (n *Node) routingLoop() {
	for message := range n.routingQueue {
		switch message["type"] {
		case control.HELLO:
			n.onHello(message["from"].(string))
		case control.LSA:
			n.onLSA(message)
		}
	}
}

func (n *Node) onHello(sender string) {
	n.mu.Lock()
	firstTime := !n.activeNeighbors[sender]
	n.activeNeighbors[sender] = true
	n.mu.Unlock()
	log.Printf("[NETWORK %s] HELLO reply from %s", n.Config.ID(), sender)
	if firstTime && !n.firstHelloSeen {
		n.firstHelloSeen = true
		log.Printf("[ROUTER %s] Waiting %s before building the LSA...", n.Config.Name, LSADelayAfterFirstHelloS)
		time.AfterFunc(LSADelayAfterFirstHelloS, n.buildAndFloodOwnLSA)
	}
}

func (n *Node) buildAndFloodOwnLSA() {
	links := map[string]int{}
	n.mu.RLock()
	for _, neighbor := range n.Config.Neighbors {
		if n.activeNeighbors[neighbor.ID()] {
			links[neighbor.ID()] = neighbor.Cost
		}
	}
	n.mu.RUnlock()
	if n.Config.AttachedHost != nil {
		links[n.Config.AttachedHost.ID()] = n.Config.AttachedHost.Cost
	}
	seq := n.store.NextSeq()
	lsa := control.BuildLSA(n.Config.ID(), seq, links, n.Config.ID())
	log.Printf("[ROUTER %s] LSA: %+v", n.Config.Name, lsa)
	n.store.Record(n.Config.ID(), seq, links)
	n.flood(lsa, "")
}

func (n *Node) onLSA(message map[string]interface{}) {
	origin := message["origin"].(string)
	seq := int(message["seq"].(float64))
	sender := message["from"].(string)
	links := map[string]int{}
	for node, cost := range message["links"].(map[string]interface{}) {
		links[node] = int(cost.(float64))
	}
	lsa := control.LSAMessage{Type: control.LSA, Origin: origin, Seq: seq, Links: links, From: sender}
	if n.store.Record(origin, seq, links) {
		log.Printf("[NETWORK %s] LSA from %s stored -> flooding onward", n.Config.ID(), origin)
		n.flood(lsa, sender)
	} else {
		log.Printf("[NETWORK %s] Ignoring LSA from %s (own or already known)", n.Config.ID(), origin)
	}
}

func (n *Node) flood(lsa control.LSAMessage, excludeID string) {
	for _, neighbor := range n.Config.Neighbors {
		if neighbor.ID() != excludeID {
			lsa.From = n.Config.ID()
			n.SendMessage(neighbor.IP, neighbor.Port, lsa)
		}
	}
}

// -- convergencia -------------------------------------------------------------

func (n *Node) convergenceTimer() {
	time.Sleep(ConvergenceWaitS)
	graph := n.store.Snapshot()
	log.Printf("[ROUTER %s] Converged. Network graph: %+v", n.Config.Name, graph)
	paths := algorithms.ShortestPaths(graph, n.Config.ID())
	log.Printf("[ROUTER %s] Shortest paths (Dijkstra):", n.Config.Name)
	for destination, result := range paths {
		log.Printf("  %s: cost %d via %s", destination, result.Cost, result.NextHop)
	}
	routes := map[string]control.Route{}
	for destination, result := range paths {
		routes[destination] = control.Route{
			Destination: destination,
			NextHopIP:   n.ipFor(result.NextHop),
			NextHopPort: n.portFor(result.NextHop),
			Cost:        result.Cost,
		}
	}
	n.mu.Lock()
	n.routes = routes
	n.mu.Unlock()
	if err := control.WriteRoutingTable(n.CSVPath, routes); err != nil {
		log.Printf("[NETWORK %s] error escribiendo tabla de ruteo: %v", n.Config.ID(), err)
		return
	}
	log.Printf("[NETWORK %s] Routing table written to %s", n.Config.ID(), n.CSVPath)
	log.Printf("[ROUTER %s] Done (Ctrl-C to stop).", n.Config.Name)
}

func (n *Node) portFor(nodeID string) int {
	for _, neighbor := range n.Config.Neighbors {
		if neighbor.ID() == nodeID {
			return neighbor.Port
		}
	}
	if n.Config.AttachedHost != nil && n.Config.AttachedHost.ID() == nodeID {
		return n.Config.AttachedHost.Port
	}
	return n.Config.Port
}

func (n *Node) ipFor(nodeID string) string {
	for _, neighbor := range n.Config.Neighbors {
		if neighbor.ID() == nodeID {
			return neighbor.IP
		}
	}
	if n.Config.AttachedHost != nil && n.Config.AttachedHost.ID() == nodeID {
		return n.Config.AttachedHost.IP
	}
	return n.Config.IP
}

// -- plano de datos (goroutine de forwarding) ------------------------------------

func (n *Node) forwardingLoop() {
	for envelope := range n.forwardingQueue {
		send := func(ip string, port int, message interface{}) { n.SendMessage(ip, port, message) }
		n.mu.RLock()
		routes := n.routes
		n.mu.RUnlock()
		if err := forwarding.Forward(envelope, routes, send); err != nil {
			log.Printf("[NETWORK %s] %v", n.Config.ID(), err)
		}
	}
}
