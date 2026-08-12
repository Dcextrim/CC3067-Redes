// Package forwarding: decodifica DATA, consulta la tabla de ruteo y reenvia.
//
// Hamming(7,4) por bloques protege el frame {from, to, msg} completo en cada
// salto (el frame se parte en bloques de 4 bits, cada uno codificado a 7). Un
// router intermedio corrige y deserializa el frame entero (asi lo exige
// Hamming), pero solo LEE el campo "to" para decidir hacia donde reenviar --
// nunca actua sobre "msg". Unicamente el destino final interpreta "msg".
package forwarding

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cc3067/lab3/internal/router/algorithms"
	"cc3067/lab3/internal/router/codec"
	"cc3067/lab3/internal/router/control"
)

// DecodeFrame corrige errores sobre TODOS los bits del frame recibido.
func DecodeFrame(message control.DataEnvelope) (string, error) {
	result := algorithms.HammingDecode(message.Bits, message.Len)
	if !result.Valid {
		return "", fmt.Errorf("frame DATA invalido: %s", result.Error)
	}
	return result.DataBits, nil
}

// DecodePayload deserializa el frame ya corregido; usado por routers y hosts.
func DecodePayload(message control.DataEnvelope) (control.Payload, error) {
	dataBits, err := DecodeFrame(message)
	if err != nil {
		return control.Payload{}, err
	}
	text, err := codec.DecodificarMensaje(dataBits)
	if err != nil {
		return control.Payload{}, err
	}
	var payload control.Payload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return control.Payload{}, err
	}
	return payload, nil
}

// EncodeEnvelope serializa {from,to,msg} y aplica Hamming(7,4) al frame completo.
func EncodeEnvelope(payload control.Payload) (control.DataEnvelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return control.DataEnvelope{}, err
	}
	dataBits, err := codec.CodificarMensaje(string(raw))
	if err != nil {
		return control.DataEnvelope{}, err
	}
	encoded, err := algorithms.HammingEncode(dataBits)
	if err != nil {
		return control.DataEnvelope{}, err
	}
	return control.DataEnvelope{Type: control.DATA, Len: len(dataBits), Bits: encoded}, nil
}

// ApplyNoise simula el canal no confiable del Lab 2 sobre toda la trama DATA
// ya protegida. Tambien los bits de paridad quedan expuestos a posibles flips.
func ApplyNoise(message control.DataEnvelope, probability float64, source *rand.Rand) (control.DataEnvelope, int, error) {
	bits, flips, err := algorithms.AplicarRuido(message.Bits, probability, source)
	if err != nil {
		return control.DataEnvelope{}, 0, err
	}
	message.Bits = bits
	return message, flips, nil
}

// SendFunc envia un mensaje ya serializable por socket hacia ip:puerto.
type SendFunc func(ip string, port int, message interface{})

// ForwardUsingRoutingTable cumple el contrato del laboratorio: consulta el
// archivo <nodo>_tabla_enrutamiento.csv para cada trama DATA y usa la ruta
// vigente escrita por el plano de control.
func ForwardUsingRoutingTable(message control.DataEnvelope, tablePath string, send SendFunc) error {
	routes, err := control.ReadRoutingTable(tablePath)
	if err != nil {
		return fmt.Errorf("no se pudo consultar la tabla de ruteo %s: %w", tablePath, err)
	}
	return Forward(message, routes, send)
}

// Forward deserializa solo "to", consulta la tabla, re-codifica y reenvia.
func Forward(message control.DataEnvelope, routes map[string]control.Route, send SendFunc) error {
	dataBits, err := DecodeFrame(message)
	if err != nil {
		return err
	}
	text, err := codec.DecodificarMensaje(dataBits)
	if err != nil {
		return err
	}
	var payload control.Payload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return err
	}

	route, ok := routes[payload.To]
	if !ok {
		return fmt.Errorf("sin ruta hacia %s; descartando DATA", payload.To)
	}

	encoded, err := algorithms.HammingEncode(dataBits)
	if err != nil {
		return err
	}
	envelope := control.DataEnvelope{Type: control.DATA, Len: len(dataBits), Bits: encoded}
	send(route.NextHopIP, route.NextHopPort, envelope)
	return nil
}
