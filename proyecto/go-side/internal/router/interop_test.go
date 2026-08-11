package router

import (
	"encoding/json"
	"os/exec"
	"testing"

	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

// TestGoDecodesPythonEnvelope confirma que un frame DATA armado por el lado
// Python (Hamming sobre el frame completo) se decodifica igual en Go. Se
// salta si no hay un interprete de Python en el PATH. Ver
// tests/integration_cross_language.py para la contraparte que confirma lo
// inverso (Python decodifica un frame armado por Go).
func TestGoDecodesPythonEnvelope(t *testing.T) {
	python := findPython(t)

	script := `
import json, sys
sys.path.insert(0, "python-side")
from router.forwarding.network import encode_envelope
print(json.dumps(encode_envelope({"from": "A", "to": "B", "msg": "interop"})))
`
	cmd := exec.Command(python, "-c", script)
	cmd.Dir = "../../.." // go-side/internal/router -> proyecto/
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("no se pudo ejecutar el script Python: %v", err)
	}

	var envelope control.DataEnvelope
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("salida invalida de Python: %s (%v)", out, err)
	}

	payload, err := forwarding.DecodePayload(envelope)
	if err != nil {
		t.Fatalf("Go no pudo decodificar el frame de Python: %v", err)
	}
	if payload.From != "A" || payload.To != "B" || payload.Msg != "interop" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
}

func findPython(t *testing.T) string {
	for _, candidate := range []string{"python3", "python"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Skip("python3 no disponible en PATH; se omite la prueba de interoperabilidad")
	return ""
}
