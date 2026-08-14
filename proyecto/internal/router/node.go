package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"cc3067/lab3/internal/router/algorithms"
	"cc3067/lab3/internal/router/control"
	"cc3067/lab3/internal/router/forwarding"
)

const (
	HelloIntervalS           = 10 * time.Second
	LSADelayAfterFirstHelloS = 5 * time.Second
	ConvergenceWaitS         = 30 * time.Second

	// NeighborTimeoutS: si no llega HELLO de un vecino activo en este lapso
	// (3 ciclos de HELLO), se marca como caido y se regenera el LSA propio.
	NeighborTimeoutS       = 3 * HelloIntervalS
	NeighborCheckIntervalS = 5 * time.Second

	// RouteRecomputeIntervalS es el respaldo periodico: ademas de recalcular
	// de inmediato cuando cambia el grafo (nuevo LSA, vecino caido/recuperado),
	// se revisa con esta cadencia por si algun evento se perdio.
	RouteRecomputeIntervalS = 15 * time.Second

	// LSARebuildDebounceS evita floodear un LSA nuevo por cada cambio de
	// vecino si varios ocurren juntos (p.ej. varios vecinos caen a la vez).
	LSARebuildDebounceS = 3 * time.Second
)

// Node es un router: goroutine de escucha + goroutine de routing + goroutine
// de forwarding corriendo en paralelo (ver seccion 3.3 del enunciado).
type Node struct {
	Config  NodeConfig
	CSVPath string

	// Overrides opcionales de temporizacion, solo para pruebas (por defecto
	// se usan las constantes de arriba, que son las exigidas por el enunciado).
	HelloInterval           time.Duration
	LSADelayAfterFirstHello time.Duration
	ConvergenceWait         time.Duration
	NeighborTimeout         time.Duration
	NeighborCheckInterval   time.Duration
	RouteRecomputeInterval  time.Duration
	LSARebuildDebounce      time.Duration

	store *control.LinkStateStore

	// mu protege todo el estado mutable compartido entre goroutines:
	// activeNeighbors/lastHelloAt (routingLoop + neighborWatchdog), routes
	// (convergenceLoop + forwardingLoop), y los flags de coordinacion.
	mu              sync.RWMutex
	recomputeMu     sync.Mutex
	noiseMu         sync.Mutex
	noiseSource     *rand.Rand
	activeNeighbors map[string]bool
	lastHelloAt     map[string]time.Time
	routes          map[string]control.Route
	firstHelloSeen  bool
	converged       bool
	lastLSARebuild  time.Time
	listener        net.Listener
	stop            chan struct{}
	stopOnce        sync.Once

	routingQueue    chan map[string]interface{}
	forwardingQueue chan control.DataEnvelope
}

// NewNode crea un nodo listo para arrancar con Start/RunForever.
func NewNode(config NodeConfig, csvPath string) *Node {
	if csvPath == "" {
		csvPath = fmt.Sprintf("%s_tabla_enrutamiento.csv", config.Name)
	}
	return &Node{
		Config:                  config,
		CSVPath:                 csvPath,
		HelloInterval:           HelloIntervalS,
		LSADelayAfterFirstHello: LSADelayAfterFirstHelloS,
		ConvergenceWait:         ConvergenceWaitS,
		NeighborTimeout:         NeighborTimeoutS,
		NeighborCheckInterval:   NeighborCheckIntervalS,
		RouteRecomputeInterval:  RouteRecomputeIntervalS,
		LSARebuildDebounce:      LSARebuildDebounceS,
		store:                   control.NewLinkStateStore(),
		noiseSource:             rand.New(rand.NewSource(time.Now().UnixNano() + int64(config.Port))),
		activeNeighbors:         map[string]bool{},
		lastHelloAt:             map[string]time.Time{},
		routes:                  map[string]control.Route{},
		routingQueue:            make(chan map[string]interface{}, 64),
		forwardingQueue:         make(chan control.DataEnvelope, 64),
		stop:                    make(chan struct{}),
	}
}

// Start lanza las goroutines del nodo y retorna de inmediato.
func (n *Node) Start() {
	go n.listenLoop()
	go n.helloLoop()
	go n.routingLoop()
	go n.forwardingLoop()
	go n.neighborWatchdog()
	go n.convergenceLoop()
}

// RunForever arranca el nodo y bloquea el proceso principal.
func (n *Node) RunForever() {
	n.Start()
	<-n.stop
}

// Stop detiene los loops y cierra el listener. Es idempotente y permite que
// las pruebas liberen sockets/goroutines sin esperar a que termine el proceso.
func (n *Node) Stop() {
	n.stopOnce.Do(func() {
		close(n.stop)
		n.mu.Lock()
		listener := n.listener
		n.listener = nil
		n.mu.Unlock()
		if listener != nil {
			_ = listener.Close()
		}
	})
}

func (n *Node) stopped() bool {
	select {
	case <-n.stop:
		return true
	default:
		return false
	}
}

// SendMessage abre una conexion TCP corta, envia un mensaje JSON y cierra.
func (n *Node) SendMessage(ip string, port int, message interface{}) {
	if n.stopped() {
		return
	}
	address := NodeID(ip, port)
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
	address := NodeID(n.Config.IP, n.Config.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("[NETWORK %s] no se pudo escuchar en %s: %v", n.Config.ID(), address, err)
		n.Stop()
		return
	}
	n.mu.Lock()
	if n.stopped() {
		n.mu.Unlock()
		_ = listener.Close()
		return
	}
	n.listener = listener
	n.mu.Unlock()
	log.Printf("[NETWORK %s] Listening", n.Config.ID())
	for {
		conn, err := listener.Accept()
		if err != nil {
			if n.stopped() {
				return
			}
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
			select {
			case n.forwardingQueue <- envelope:
			case <-n.stop:
			}
		}
		return
	}
	select {
	case n.routingQueue <- generic:
	case <-n.stop:
	}
}

// -- plano de control (goroutine de routing) --------------------------------

func (n *Node) helloLoop() {
	for {
		for _, neighbor := range n.Config.Neighbors {
			n.SendMessage(neighbor.IP, neighbor.Port, control.BuildHello(n.Config.ID()))
		}
		timer := time.NewTimer(n.HelloInterval)
		select {
		case <-timer.C:
		case <-n.stop:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func (n *Node) routingLoop() {
	for {
		select {
		case message := <-n.routingQueue:
			switch message["type"] {
			case control.HELLO:
				n.onHello(message["from"].(string))
			case control.LSA:
				n.onLSA(message)
			}
		case <-n.stop:
			return
		}
	}
}

func (n *Node) onHello(sender string) {
	n.mu.Lock()
	n.lastHelloAt[sender] = time.Now()
	firstTime := !n.activeNeighbors[sender]
	n.activeNeighbors[sender] = true
	isVeryFirstHello := !n.firstHelloSeen
	if isVeryFirstHello {
		n.firstHelloSeen = true
	}
	n.mu.Unlock()
	log.Printf("[NETWORK %s] HELLO reply from %s", n.Config.ID(), sender)
	if !firstTime {
		return
	}
	// Este vecino especifico recien aparece (primera vez o recuperado): puede
	// no conocer LSA que este nodo ya acepto hace tiempo y que nadie ha tenido
	// motivo de volver a floodear. Sin este resync, un vecino que se reincorpora
	// se queda ciego ante partes de la red (routers y hosts adjuntos) que no
	// cambiaron de estado desde su ultima flood original.
	n.syncLSADatabaseTo(sender)
	if isVeryFirstHello {
		log.Printf("[ROUTER %s] Waiting %s before building the LSA...", n.Config.Name, n.LSADelayAfterFirstHello)
		time.AfterFunc(n.LSADelayAfterFirstHello, n.buildAndFloodOwnLSA)
		return
	}
	// Un vecino que ya habia caido volvio a responder: no es el primer HELLO
	// del nodo, pero si cambia la topologia -> hay que avisar con un LSA nuevo.
	n.requestLSARebuild()
}

// syncLSADatabaseTo reenvia directamente a neighborID cada LSA que este nodo
// ya tiene guardada (ver LinkStateStore.AllRecords). El receptor las procesa
// con su onLSA normal: las que ya conoce se descartan por seq, las nuevas se
// aceptan y siguen propagandose como cualquier LSA recien aprendida.
func (n *Node) syncLSADatabaseTo(neighborID string) {
	var target *Neighbor
	for i := range n.Config.Neighbors {
		if n.Config.Neighbors[i].ID() == neighborID {
			target = &n.Config.Neighbors[i]
			break
		}
	}
	if target == nil {
		return
	}
	for _, lsa := range n.store.AllRecords() {
		lsa.From = n.Config.ID()
		n.SendMessage(target.IP, target.Port, lsa)
	}
}

// neighborWatchdog vigila que los vecinos activos sigan enviando HELLO; si
// uno deja de hacerlo por NeighborTimeout, se marca como caido y se pide un
// LSA nuevo (la topologia cambio, igual que si el vecino nunca hubiera estado ahi).
func (n *Node) neighborWatchdog() {
	ticker := time.NewTicker(n.NeighborCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-n.stop:
			return
		case <-ticker.C:
		}
		changed := false
		n.mu.Lock()
		for _, neighbor := range n.Config.Neighbors {
			id := neighbor.ID()
			if !n.activeNeighbors[id] {
				continue
			}
			if time.Since(n.lastHelloAt[id]) > n.NeighborTimeout {
				n.activeNeighbors[id] = false
				changed = true
				log.Printf("[ROUTER %s] Neighbor %s expired (sin HELLO en %s)", n.Config.Name, id, n.NeighborTimeout)
			}
		}
		n.mu.Unlock()
		if changed {
			n.requestLSARebuild()
		}
	}
}

// requestLSARebuild reconstruye y floodea el LSA propio, respetando un
// debounce minimo para no inundar la red si varios vecinos cambian a la vez.
func (n *Node) requestLSARebuild() {
	n.mu.Lock()
	elapsed := time.Since(n.lastLSARebuild)
	if elapsed < n.LSARebuildDebounce {
		wait := n.LSARebuildDebounce - elapsed
		n.mu.Unlock()
		time.AfterFunc(wait, n.buildAndFloodOwnLSA)
		return
	}
	n.mu.Unlock()
	n.buildAndFloodOwnLSA()
}

func (n *Node) buildAndFloodOwnLSA() {
	if n.stopped() {
		return
	}
	links := map[string]int{}
	n.mu.Lock()
	n.lastLSARebuild = time.Now()
	for _, neighbor := range n.Config.Neighbors {
		if n.activeNeighbors[neighbor.ID()] {
			links[neighbor.ID()] = neighbor.Cost
		}
	}
	n.mu.Unlock()
	if n.Config.AttachedHost != nil {
		links[n.Config.AttachedHost.ID()] = n.Config.AttachedHost.Cost
	}
	seq := n.store.NextSeq()
	lsa := control.BuildLSA(n.Config.ID(), seq, links, n.Config.ID())
	log.Printf("[ROUTER %s] LSA: %+v", n.Config.Name, lsa)
	n.store.Record(n.Config.ID(), seq, links)
	n.flood(lsa, "")
	n.maybeRecomputeRoutes()
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
		n.maybeRecomputeRoutes()
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

// convergenceLoop espera la convergencia inicial exigida por el enunciado
// (30s) y calcula la primera tabla de ruteo; despues sigue recalculando cada
// vez que cambia el grafo (maybeRecomputeRoutes) y, como respaldo, en cada
// tick de RouteRecomputeInterval por si algun evento no disparo el recalculo.
func (n *Node) convergenceLoop() {
	timer := time.NewTimer(n.ConvergenceWait)
	select {
	case <-timer.C:
	case <-n.stop:
		if !timer.Stop() {
			<-timer.C
		}
		return
	}
	n.mu.Lock()
	n.converged = true
	n.mu.Unlock()
	n.recomputeRoutes()
	log.Printf("[ROUTER %s] Converged. Recomputando rutas ante cada cambio de topologia.", n.Config.Name)

	ticker := time.NewTicker(n.RouteRecomputeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			n.recomputeRoutes()
		case <-n.stop:
			return
		}
	}
}

// maybeRecomputeRoutes recalcula de inmediato si ya paso la convergencia
// inicial; antes de eso, la primera tabla la produce convergenceLoop segun
// el tiempo de espera exigido por el enunciado.
func (n *Node) maybeRecomputeRoutes() {
	n.mu.RLock()
	converged := n.converged
	n.mu.RUnlock()
	if converged {
		n.recomputeRoutes()
	}
}

func (n *Node) recomputeRoutes() {
	n.recomputeMu.Lock()
	defer n.recomputeMu.Unlock()
	if n.stopped() {
		return
	}
	graph := n.store.Snapshot()
	paths := algorithms.ShortestPaths(graph, n.Config.ID())
	routes := map[string]control.Route{}
	for destination, result := range paths {
		routes[destination] = control.Route{
			Destination: destination,
			NextHopIP:   n.ipFor(result.NextHop),
			NextHopPort: n.portFor(result.NextHop),
			Cost:        result.Cost,
		}
	}

	n.mu.RLock()
	changed := !routesEqual(n.routes, routes)
	n.mu.RUnlock()
	if !changed {
		return
	}

	log.Printf("[ROUTER %s] Shortest paths (Dijkstra):", n.Config.Name)
	for destination, result := range paths {
		log.Printf("  %s: cost %d via %s", destination, result.Cost, result.NextHop)
	}
	if err := control.WriteRoutingTable(n.CSVPath, routes); err != nil {
		log.Printf("[NETWORK %s] error escribiendo tabla de ruteo: %v", n.Config.ID(), err)
		return
	}
	// Publicar la ruta solo despues de que el CSV completo ya es visible. Asi,
	// cualquier consumidor que observe n.routes tambien puede leer la tabla.
	n.mu.Lock()
	n.routes = routes
	n.mu.Unlock()
	log.Printf("[NETWORK %s] Routing table written to %s", n.Config.ID(), n.CSVPath)
}

func routesEqual(a, b map[string]control.Route) bool {
	if len(a) != len(b) {
		return false
	}
	for destination, routeA := range a {
		routeB, ok := b[destination]
		if !ok || routeA != routeB {
			return false
		}
	}
	return true
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
	for {
		select {
		case envelope := <-n.forwardingQueue:
			send := func(ip string, port int, message interface{}) {
				outgoing, ok := message.(control.DataEnvelope)
				if !ok {
					log.Printf("[NETWORK %s] forwarding produjo un mensaje DATA invalido", n.Config.ID())
					return
				}
				n.noiseMu.Lock()
				noisy, flips, err := forwarding.ApplyNoise(outgoing, n.Config.NoiseProbability, n.noiseSource)
				n.noiseMu.Unlock()
				if err != nil {
					log.Printf("[NETWORK %s] no se pudo aplicar ruido: %v", n.Config.ID(), err)
					return
				}
				if flips > 0 {
					log.Printf("[NETWORK %s] ruido DATA hacia %s:%d: %d bits alterados (p=%g)", n.Config.ID(), ip, port, flips, n.Config.NoiseProbability)
				}
				log.Printf("[NETWORK %s] DATA reenviada -> siguiente salto %s:%d", n.Config.ID(), ip, port)
				n.SendMessage(ip, port, noisy)
			}
			if err := forwarding.ForwardUsingRoutingTable(envelope, n.CSVPath, send); err != nil {
				log.Printf("[NETWORK %s] %v", n.Config.ID(), err)
			}
		case <-n.stop:
			return
		}
	}
}
