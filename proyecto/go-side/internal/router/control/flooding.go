package control

import "sync"

type seenKey struct {
	origin string
	seq    int
}

// LinkStateStore es el estado compartido del grafo entre el hilo/goroutine
// de routing y las conexiones entrantes que reciben LSA.
type LinkStateStore struct {
	SelfID string

	mu    sync.Mutex
	graph map[string]map[string]int
	seen  map[seenKey]bool
	seq   int
}

// NewLinkStateStore crea el estado vacio para un nodo.
func NewLinkStateStore(selfID string) *LinkStateStore {
	return &LinkStateStore{
		SelfID: selfID,
		graph:  map[string]map[string]int{},
		seen:   map[seenKey]bool{},
	}
}

// NextSeq incrementa y retorna el numero de secuencia para un LSA propio.
func (s *LinkStateStore) NextSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

// Record registra un LSA si (origin, seq) es nuevo; retorna false si ya se vio.
func (s *LinkStateStore) Record(origin string, seq int, links map[string]int) bool {
	key := seenKey{origin, seq}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return false
	}
	s.seen[key] = true
	copied := make(map[string]int, len(links))
	for k, v := range links {
		copied[k] = v
	}
	s.graph[origin] = copied
	return true
}

// Snapshot retorna una copia del grafo actual, segura para usar con Dijkstra.
func (s *LinkStateStore) Snapshot() map[string]map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := make(map[string]map[string]int, len(s.graph))
	for node, links := range s.graph {
		copied := make(map[string]int, len(links))
		for k, v := range links {
			copied[k] = v
		}
		snapshot[node] = copied
	}
	return snapshot
}
