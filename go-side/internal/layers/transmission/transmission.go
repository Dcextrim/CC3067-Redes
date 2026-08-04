package transmission

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

const (
	Version      = byte(1)
	HeaderSize   = 16
	MaxFrameBits = 8 * 1024 * 1024
)

var magic = [4]byte{'C', 'C', '6', '7'}

var algorithmIDs = map[string]byte{"hamming": 1, "crc32": 2}
var idAlgorithms = map[byte]string{1: "hamming", 2: "crc32"}

type ReceivedFrame struct {
	Algorithm        string
	MessageBitLength int
	FrameBits        string
}

type Layer struct {
	connection net.Conn
	sendMutex  sync.Mutex
}

func New(connection net.Conn) *Layer {
	return &Layer{connection: connection}
}

func packBits(bits string) ([]byte, error) {
	for _, bit := range bits {
		if bit != '0' && bit != '1' {
			return nil, fmt.Errorf("la cadena solo puede contener bits 0 y 1")
		}
	}
	padded := bits + strings.Repeat("0", (8-len(bits)%8)%8)
	result := make([]byte, len(padded)/8)
	for index := range result {
		for _, bit := range padded[index*8 : index*8+8] {
			result[index] <<= 1
			result[index] |= byte(bit - '0')
		}
	}
	return result, nil
}

func unpackBits(data []byte, bitLength int) (string, error) {
	if bitLength > len(data)*8 {
		return "", fmt.Errorf("el cuerpo no contiene suficientes bits")
	}
	var builder strings.Builder
	builder.Grow(len(data) * 8)
	for _, value := range data {
		builder.WriteString(fmt.Sprintf("%08b", value))
	}
	return builder.String()[:bitLength], nil
}

func (layer *Layer) EnviarInformacion(algorithm string, messageBitLength int, frameBits string) error {
	algorithmID, ok := algorithmIDs[algorithm]
	if !ok {
		return fmt.Errorf("algoritmo no soportado")
	}
	if messageBitLength < 0 || uint64(messageBitLength) > uint64(^uint32(0)) {
		return fmt.Errorf("longitud de mensaje fuera de rango")
	}
	if len(frameBits) > MaxFrameBits {
		return fmt.Errorf("trama demasiado grande")
	}
	body, err := packBits(frameBits)
	if err != nil {
		return err
	}
	header := make([]byte, HeaderSize)
	copy(header[0:4], magic[:])
	header[4] = Version
	header[5] = algorithmID
	binary.BigEndian.PutUint16(header[6:8], 0)
	binary.BigEndian.PutUint32(header[8:12], uint32(messageBitLength))
	binary.BigEndian.PutUint32(header[12:16], uint32(len(frameBits)))

	layer.sendMutex.Lock()
	defer layer.sendMutex.Unlock()
	packet := append(header, body...)
	for len(packet) > 0 {
		written, writeErr := layer.connection.Write(packet)
		if writeErr != nil {
			return writeErr
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		packet = packet[written:]
	}
	return nil
}

func (layer *Layer) RecibirInformacion() (*ReceivedFrame, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(layer.connection, header); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	if string(header[0:4]) != string(magic[:]) || header[4] != Version || binary.BigEndian.Uint16(header[6:8]) != 0 {
		return nil, fmt.Errorf("encabezado de transmision invalido")
	}
	algorithm, ok := idAlgorithms[header[5]]
	if !ok {
		return nil, fmt.Errorf("identificador de algoritmo desconocido")
	}
	messageLength := int(binary.BigEndian.Uint32(header[8:12]))
	frameLength := int(binary.BigEndian.Uint32(header[12:16]))
	if frameLength > MaxFrameBits {
		return nil, fmt.Errorf("trama recibida demasiado grande")
	}
	body := make([]byte, (frameLength+7)/8)
	if _, err := io.ReadFull(layer.connection, body); err != nil {
		return nil, err
	}
	bits, err := unpackBits(body, frameLength)
	if err != nil {
		return nil, err
	}
	return &ReceivedFrame{algorithm, messageLength, bits}, nil
}
