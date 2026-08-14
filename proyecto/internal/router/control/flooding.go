package control

import "sync"

// LinkStateStore es el estado compartido del grafo entre el hilo/goroutine
// de routing y las conexiones entrantes que reciben LSA.
type LinkStateStore struct {
	mu        sync.Mutex
	graph     map[string]map[string]int
	latestSeq map[string]int
	seq       int
}

// NewLinkStateStore crea el estado vacio para un nodo.
func NewLinkStateStore() *LinkStateStore {
	return &LinkStateStore{
		graph:     map[string]map[string]int{},
		latestSeq: map[string]int{},
	}
}

// NextSeq incrementa y retorna el numero de secuencia para un LSA propio.
func (s *LinkStateStore) NextSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

// Record registra un LSA solo si su secuencia es mayor que la ultima aceptada
// para el mismo origen. Esto evita que un paquete atrasado revierta el grafo a
// un estado obsoleto despues de haber procesado un LSA mas reciente.
func (s *LinkStateStore) Record(origin string, seq int, links map[string]int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if latest, ok := s.latestSeq[origin]; ok && seq <= latest {
		return false
	}
	s.latestSeq[origin] = seq
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

// AllRecords retorna cada LSA conocida (origen, secuencia y links) tal como
// esta guardada actualmente. Sirve para re-enviarle a un vecino que recien
// aparece (primera vez o recuperado de una caida) todo lo que este nodo ya
// sabe: sin esto, ese vecino nunca aprenderia sobre routers que no tuvieron
// motivo propio para volver a floodear su LSA (p.ej. un host adjunto detras
// de un router que nunca cambio de estado) desde antes de que el vecino se
// reincorporara.
func (s *LinkStateStore) AllRecords() []LSAMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]LSAMessage, 0, len(s.graph))
	for origin, links := range s.graph {
		copied := make(map[string]int, len(links))
		for k, v := range links {
			copied[k] = v
		}
		result = append(result, LSAMessage{Type: LSA, Origin: origin, Seq: s.latestSeq[origin], Links: copied})
	}
	return result
}
