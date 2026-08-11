package router

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"cc3067/lab3/go-side/internal/router/algorithms"
	"cc3067/lab3/go-side/internal/router/codec"
	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

type canonicalMessageVector struct {
	JSON string `json:"json"`
}

type canonicalHammingVector struct {
	DataBits    string `json:"data_bits"`
	EncodedBits string `json:"encoded_bits"`
}

type canonicalDataVector struct {
	Payload      control.Payload      `json:"payload"`
	PayloadJSON  string               `json:"payload_json"`
	DataBits     string               `json:"data_bits"`
	Envelope     control.DataEnvelope `json:"envelope"`
	EnvelopeJSON string               `json:"envelope_json"`
}

type canonicalVectors struct {
	Version int                    `json:"version"`
	Hello   canonicalMessageVector `json:"hello"`
	LSA     canonicalMessageVector `json:"lsa"`
	Hamming canonicalHammingVector `json:"hamming_7_4"`
	Data    canonicalDataVector    `json:"data"`
}

func compactJSON(t *testing.T, value interface{}) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestCanonicalProtocolVectors convierte la interoperabilidad en un contrato
// portable: cualquier lenguaje puede consumir testdata/protocol_vectors.json
// sin necesitar ejecutar nuestra implementacion ni un segundo runtime.
func TestCanonicalProtocolVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/protocol_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors canonicalVectors
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Version != 1 {
		t.Fatalf("version de vectores no soportada: %d", vectors.Version)
	}

	hello := control.BuildHello("100.64.0.1:5000")
	if got := compactJSON(t, hello); got != vectors.Hello.JSON {
		t.Fatalf("HELLO canonico distinto:\n%s\n%s", got, vectors.Hello.JSON)
	}
	lsa := control.BuildLSA(
		"100.64.0.1:5000",
		7,
		map[string]int{"100.64.0.2:5001": 2, "100.64.0.3:5002": 5},
		"100.64.0.1:5000",
	)
	if got := compactJSON(t, lsa); got != vectors.LSA.JSON {
		t.Fatalf("LSA canonico distinto:\n%s\n%s", got, vectors.LSA.JSON)
	}

	encoded, err := algorithms.HammingEncode(vectors.Hamming.DataBits)
	if err != nil {
		t.Fatal(err)
	}
	if encoded != vectors.Hamming.EncodedBits {
		t.Fatalf("Hamming canonico = %s", encoded)
	}

	payloadJSON := compactJSON(t, vectors.Data.Payload)
	if payloadJSON != vectors.Data.PayloadJSON {
		t.Fatalf("payload JSON canonico distinto: %s", payloadJSON)
	}
	dataBits, err := codec.CodificarMensaje(payloadJSON)
	if err != nil {
		t.Fatal(err)
	}
	if dataBits != vectors.Data.DataBits {
		t.Fatal("los bits UTF-8 no coinciden con el vector canonico")
	}
	envelope, err := forwarding.EncodeEnvelope(vectors.Data.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if envelope != vectors.Data.Envelope {
		t.Fatal("el envelope DATA no coincide con el vector canonico")
	}
	if got := compactJSON(t, envelope); got != vectors.Data.EnvelopeJSON {
		t.Fatalf("DATA JSON canonico distinto: %s", got)
	}
	decoded, err := forwarding.DecodePayload(vectors.Data.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, vectors.Data.Payload) {
		t.Fatalf("payload decodificado = %+v", decoded)
	}
}
