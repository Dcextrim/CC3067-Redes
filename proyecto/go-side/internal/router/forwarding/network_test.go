package forwarding

import (
	"path/filepath"
	"testing"

	"cc3067/lab3/go-side/internal/router/control"
)

func TestForwardUsingRoutingTableReadsTheCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "A_tabla_enrutamiento.csv")
	payload := control.Payload{From: "A", To: "B", Msg: "hola"}
	envelope, err := EncodeEnvelope(payload)
	if err != nil {
		t.Fatal(err)
	}

	write := func(port int) {
		t.Helper()
		routes := map[string]control.Route{
			"B": {Destination: "B", NextHopIP: "127.0.0.1", NextHopPort: port, Cost: 1},
		}
		if err := control.WriteRoutingTable(path, routes); err != nil {
			t.Fatal(err)
		}
	}

	var gotPort int
	send := func(_ string, port int, _ interface{}) { gotPort = port }
	write(5001)
	if err := ForwardUsingRoutingTable(envelope, path, send); err != nil {
		t.Fatal(err)
	}
	if gotPort != 5001 {
		t.Fatalf("puerto obtenido del CSV = %d", gotPort)
	}

	write(5002)
	if err := ForwardUsingRoutingTable(envelope, path, send); err != nil {
		t.Fatal(err)
	}
	if gotPort != 5002 {
		t.Fatalf("no se consulto la version nueva del CSV: puerto = %d", gotPort)
	}
}
