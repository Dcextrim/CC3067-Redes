package control

import "testing"

func TestLinkStateStoreRejectsDuplicateAndStaleSequences(t *testing.T) {
	store := NewLinkStateStore()
	if !store.Record("A", 2, map[string]int{"B": 2}) {
		t.Fatal("el primer LSA debe aceptarse")
	}
	if store.Record("A", 2, map[string]int{"B": 20}) {
		t.Fatal("un LSA duplicado debe rechazarse")
	}
	if store.Record("A", 1, map[string]int{"B": 1}) {
		t.Fatal("un LSA atrasado debe rechazarse")
	}
	if got := store.Snapshot()["A"]["B"]; got != 2 {
		t.Fatalf("el LSA viejo revirtio el grafo: costo = %d", got)
	}
	if !store.Record("A", 3, map[string]int{"B": 3}) {
		t.Fatal("un LSA mas reciente debe aceptarse")
	}
	if got := store.Snapshot()["A"]["B"]; got != 3 {
		t.Fatalf("el grafo no se actualizo: costo = %d", got)
	}
}

func TestLinkStateStoreTracksSequencesPerOrigin(t *testing.T) {
	store := NewLinkStateStore()
	if !store.Record("A", 5, map[string]int{"B": 1}) {
		t.Fatal("no acepto el LSA de A")
	}
	if !store.Record("C", 1, map[string]int{"D": 1}) {
		t.Fatal("la secuencia de A no debe bloquear el LSA de C")
	}
}
