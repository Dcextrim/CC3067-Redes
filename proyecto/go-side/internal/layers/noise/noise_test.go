package noise

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
