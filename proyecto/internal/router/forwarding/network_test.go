package forwarding

import (
	"math/rand"
	"path/filepath"
	"testing"

	"cc3067/lab3/internal/router/control"
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

func TestApplyNoisePreservesEnvelopeMetadataAndAltersAllBitsAtProbabilityOne(t *testing.T) {
	original := control.DataEnvelope{Type: control.DATA, Len: 4, Bits: "001101"}
	noisy, flips, err := ApplyNoise(original, 1, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if flips != len(original.Bits) || noisy.Bits != "110010" {
		t.Fatalf("ruido inesperado: bits=%s flips=%d", noisy.Bits, flips)
	}
	if noisy.Type != original.Type || noisy.Len != original.Len {
		t.Fatalf("el ruido modifico metadatos DATA: %+v", noisy)
	}
}

func TestForwardCorrectsOneBitAndReencodesCleanFrame(t *testing.T) {
	payload := control.Payload{From: "A", To: "B", Msg: "corregible"}
	clean, err := EncodeEnvelope(payload)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := clean
	bits := []byte(corrupted.Bits)
	bits[2] ^= 1
	corrupted.Bits = string(bits)

	routes := map[string]control.Route{
		"B": {Destination: "B", NextHopIP: "127.0.0.1", NextHopPort: 5001, Cost: 1},
	}
	var forwarded control.DataEnvelope
	if err := Forward(corrupted, routes, func(_ string, _ int, message interface{}) {
		forwarded = message.(control.DataEnvelope)
	}); err != nil {
		t.Fatal(err)
	}
	if forwarded != clean {
		t.Fatal("el router no reenvio una trama limpia despues de corregir Hamming")
	}
	decoded, err := DecodePayload(forwarded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != payload {
		t.Fatalf("payload recibido = %+v; se esperaba %+v", decoded, payload)
	}
}
