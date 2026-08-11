package control

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRoutingTableRoundTripAndAtomicReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "A_tabla_enrutamiento.csv")
	first := map[string]Route{
		"C": {Destination: "C", NextHopIP: "127.0.0.1", NextHopPort: 5001, Cost: 5},
		"B": {Destination: "B", NextHopIP: "127.0.0.1", NextHopPort: 5001, Cost: 2},
	}
	if err := WriteRoutingTable(path, first); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadRoutingTable(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, first) {
		t.Fatalf("round trip distinto: %#v", loaded)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nB,") || strings.Index(string(raw), "\nB,") > strings.Index(string(raw), "\nC,") {
		t.Fatalf("las rutas no quedaron ordenadas: %s", raw)
	}

	second := map[string]Route{
		"D": {Destination: "D", NextHopIP: "127.0.0.1", NextHopPort: 5002, Cost: 1},
	}
	if err := WriteRoutingTable(path, second); err != nil {
		t.Fatal(err)
	}
	loaded, err = ReadRoutingTable(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, second) {
		t.Fatalf("el reemplazo no fue completo: %#v", loaded)
	}
}

func TestReadRoutingTableRejectsMalformedRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.csv")
	if err := os.WriteFile(path, []byte("destination,next_hop_ip,next_hop_port,cost\nB,127.0.0.1,nope,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRoutingTable(path); err == nil {
		t.Fatal("se esperaba error para un puerto invalido")
	}
}
