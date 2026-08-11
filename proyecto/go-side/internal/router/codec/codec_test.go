package codec

import "testing"

func TestCodecRoundTrip(t *testing.T) {
	bits, err := CodificarMensaje("A router message")
	if err != nil {
		t.Fatal(err)
	}
	if bits[:8] != "01000001" {
		t.Fatalf("A = %s", bits[:8])
	}
	message, err := DecodificarMensaje(bits)
	if err != nil || message != "A router message" {
		t.Fatalf("message=%q err=%v", message, err)
	}
}
