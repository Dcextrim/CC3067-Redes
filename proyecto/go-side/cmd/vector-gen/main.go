// Command vector-gen genera los vectores canonicos e independientes del
// lenguaje para comprobar compatibilidad con otras implementaciones.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"cc3067/lab3/go-side/internal/router/algorithms"
	"cc3067/lab3/go-side/internal/router/codec"
	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

type messageVector struct {
	JSON string `json:"json"`
}

type hammingVector struct {
	DataBits    string `json:"data_bits"`
	EncodedBits string `json:"encoded_bits"`
}

type dataVector struct {
	Payload      control.Payload      `json:"payload"`
	PayloadJSON  string               `json:"payload_json"`
	DataBits     string               `json:"data_bits"`
	Envelope     control.DataEnvelope `json:"envelope"`
	EnvelopeJSON string               `json:"envelope_json"`
}

type protocolVectors struct {
	Version     int           `json:"version"`
	Description string        `json:"description"`
	Hello       messageVector `json:"hello"`
	LSA         messageVector `json:"lsa"`
	Hamming     hammingVector `json:"hamming_7_4"`
	Data        dataVector    `json:"data"`
}

func mustJSON(value interface{}) string {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func main() {
	output := flag.String("output", "internal/router/testdata/protocol_vectors.json", "archivo JSON de salida")
	flag.Parse()

	hello := control.BuildHello("100.64.0.1:5000")
	lsa := control.BuildLSA(
		"100.64.0.1:5000",
		7,
		map[string]int{"100.64.0.2:5001": 2, "100.64.0.3:5002": 5},
		"100.64.0.1:5000",
	)
	payload := control.Payload{From: "A", To: "B", Msg: "interop"}
	payloadJSON := mustJSON(payload)
	dataBits, err := codec.CodificarMensaje(payloadJSON)
	if err != nil {
		panic(err)
	}
	envelope, err := forwarding.EncodeEnvelope(payload)
	if err != nil {
		panic(err)
	}
	hammingBits, err := algorithms.HammingEncode("1011")
	if err != nil {
		panic(err)
	}

	vectors := protocolVectors{
		Version:     1,
		Description: "Vectores canonicos CC3067 Lab 3; JSON UTF-8 compacto y Hamming(7,4) por bloques con paridad par.",
		Hello:       messageVector{JSON: mustJSON(hello)},
		LSA:         messageVector{JSON: mustJSON(lsa)},
		Hamming:     hammingVector{DataBits: "1011", EncodedBits: hammingBits},
		Data: dataVector{
			Payload:      payload,
			PayloadJSON:  payloadJSON,
			DataBits:     dataBits,
			Envelope:     envelope,
			EnvelopeJSON: mustJSON(envelope),
		},
	}
	raw, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		panic(err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("Vectores escritos en %s\n", *output)
}
