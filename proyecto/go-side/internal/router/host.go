package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

// SendViaGateway arma el DATA inicial y lo entrega al router-gateway local.
// selfID / toID son identificadores "ip:puerto" (ver NodeID), iguales a los
// que el router uso para este host en su LSA.
func SendViaGateway(selfID, gatewayIP string, gatewayPort int, toID, text string) error {
	envelope, err := forwarding.EncodeEnvelope(control.Payload{From: selfID, To: toID, Msg: text})
	if err != nil {
		return err
	}
	address := fmt.Sprintf("%s:%d", gatewayIP, gatewayPort)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	raw, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(raw, '\n'))
	return err
}

// RunServer escucha DATA que el router-gateway local ya resolvio hasta este host.
func RunServer(listenIP string, listenPort int, onMessage func(control.Payload)) error {
	address := fmt.Sprintf("%s:%d", listenIP, listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			line, err := bufio.NewReader(c).ReadString('\n')
			if err != nil && line == "" {
				return
			}
			var envelope control.DataEnvelope
			if err := json.Unmarshal([]byte(line), &envelope); err != nil {
				return
			}
			payload, err := forwarding.DecodePayload(envelope)
			if err == nil {
				onMessage(payload)
			}
		}(conn)
	}
}
