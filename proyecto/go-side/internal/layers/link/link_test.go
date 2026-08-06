package link

import "testing"

func TestLinkHammingCorrection(t *testing.T) {
	data := "01000001"
	frame, _ := CalcularIntegridad(data, Hamming)
	corrupted := []byte(frame)
	corrupted[4] ^= 1
	result := VerificarIntegridad(string(corrupted), Hamming, len(data))
	if !result.OK || !result.Corrected || result.MessageBits != data {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}
