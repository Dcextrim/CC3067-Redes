package algorithms

import (
	"math/rand"
	"testing"
)

func TestNoiseAndFraction(t *testing.T) {
	probability, err := ParseProbability("1/100")
	if err != nil || probability != 0.01 {
		t.Fatalf("probabilidad=%f err=%v", probability, err)
	}
	output, flips, err := AplicarRuido("001101", 1, rand.New(rand.NewSource(1)))
	if err != nil || output != "110010" || flips != 6 {
		t.Fatalf("output=%s flips=%d err=%v", output, flips, err)
	}
}

func TestNoiseRejectsInvalidInputs(t *testing.T) {
	if _, err := ParseProbability("1/0"); err == nil {
		t.Fatal("se esperaba error para una fraccion con denominador cero")
	}
	if _, _, err := AplicarRuido("0101", 0.1, nil); err == nil {
		t.Fatal("se esperaba error sin fuente aleatoria")
	}
	if _, _, err := AplicarRuido("01x1", 0.1, rand.New(rand.NewSource(1))); err == nil {
		t.Fatal("se esperaba error para una cadena que no contiene solo bits")
	}
}
