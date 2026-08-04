package presentation

import "testing"

func TestPresentationRoundTrip(t *testing.T) {
	bits, err := CodificarMensaje("A bank message")
	if err != nil {
		t.Fatal(err)
	}
	if bits[:8] != "01000001" {
		t.Fatalf("A = %s", bits[:8])
	}
	message, err := DecodificarMensaje(bits)
	if err != nil || message != "A bank message" {
		t.Fatalf("message=%q err=%v", message, err)
	}
}
