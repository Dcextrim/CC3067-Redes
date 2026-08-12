package commandui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestPromptNoiseDefaultsToCompatibleMode(t *testing.T) {
	var output bytes.Buffer
	probability, err := PromptNoiseProbability(bufio.NewReader(strings.NewReader("\n")), &output, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if probability != 0 {
		t.Fatalf("la opcion predeterminada debe ser sin ruido; se obtuvo %g", probability)
	}
}

func TestPromptNoiseCanEnableDefaultOrCustomProbability(t *testing.T) {
	tests := []struct {
		input      string
		configured float64
		want       float64
	}{
		{input: "s\n\n", want: 0.001},
		{input: "si\n1/100\n", want: 0.01},
		{input: "s\n\n", configured: 0.005, want: 0.005},
	}
	for _, test := range tests {
		var output bytes.Buffer
		got, err := PromptNoiseProbability(bufio.NewReader(strings.NewReader(test.input)), &output, test.configured)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("entrada %q: probabilidad=%g; se esperaba %g", test.input, got, test.want)
		}
	}
}

func TestPromptNoiseRepeatsInvalidAnswers(t *testing.T) {
	var output bytes.Buffer
	input := bufio.NewReader(strings.NewReader("talvez\ns\n2\n1/1000\n"))
	probability, err := PromptNoiseProbability(input, &output, 0)
	if err != nil {
		t.Fatal(err)
	}
	if probability != 0.001 {
		t.Fatalf("probabilidad=%g", probability)
	}
	if !strings.Contains(output.String(), "Responda s") || !strings.Contains(output.String(), "Valor invalido") {
		t.Fatalf("no se explicaron las entradas invalidas: %s", output.String())
	}
}
