// Package control implementa el plano de control: HELLO, LSA, flooding y tabla de ruteo.
package control

// Tipos de mensaje del protocolo (docs/protocol.md).
const (
	HELLO = "HELLO"
	LSA   = "LSA"
	DATA  = "DATA"
)

// Hello es el mensaje periodico que confirma que un vecino esta vivo.
type Hello struct {
	Type string `json:"type"`
	From string `json:"from"`
}

// BuildHello arma un HELLO desde el identificador propio (ip:puerto).
func BuildHello(fromID string) Hello { return Hello{Type: HELLO, From: fromID} }

// LSAMessage es el Link State Advertisement que se inunda a toda la red.
type LSAMessage struct {
	Type  string         `json:"type"`
	Origin string        `json:"origin"`
	Seq   int            `json:"seq"`
	Links map[string]int `json:"links"`
	From  string         `json:"from"`
}

// BuildLSA arma un LSA propio; origin y from inician iguales al emisor.
func BuildLSA(origin string, seq int, links map[string]int, fromID string) LSAMessage {
	return LSAMessage{Type: LSA, Origin: origin, Seq: seq, Links: links, From: fromID}
}

// DataEnvelope transporta el frame {from,to,msg} protegido con Hamming(7,4)
// completo (ver forwarding.Forward): un router intermedio decodifica TODO
// el frame pero solo lee "to"; unicamente el destino final usa "msg".
type DataEnvelope struct {
	Type string `json:"type"`
	Len  int    `json:"len"`
	Bits string `json:"bits"`
}

// Payload es el contenido logico protegido dentro de un DataEnvelope.
type Payload struct {
	From string `json:"from"`
	To   string `json:"to"`
	Msg  string `json:"msg"`
}
