package algorithms

import (
	"fmt"
	"strings"
)

// CRC32ReflectedPolynomial es la forma reflejada del polinomio IEEE 0x04C11DB7.
const CRC32ReflectedPolynomial uint32 = 0xEDB88320

func paddedBytes(bits string) ([]byte, error) {
	// El convenio del protocolo rellena hasta 32 bits y despues a un octeto.
	if err := validateBits(bits); err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString(bits)
	if len(bits) <= 32 {
		builder.WriteString(strings.Repeat("0", 32-len(bits)))
	}
	padded := builder.String()
	if remainder := len(padded) % 8; remainder != 0 {
		padded += strings.Repeat("0", 8-remainder)
	}
	result := make([]byte, len(padded)/8)
	for index := range result {
		var value byte
		for _, bit := range padded[index*8 : index*8+8] {
			value <<= 1
			value |= byte(bit - '0')
		}
		result[index] = value
	}
	return result, nil
}

// CRC32Calculate implementa CRC-32/ISO-HDLC con 0xEDB88320.
func CRC32Calculate(bits string) (uint32, error) {
	data, err := paddedBytes(bits)
	if err != nil {
		return 0, err
	}
	crc := uint32(0xFFFFFFFF)
	for _, value := range data {
		crc ^= uint32(value)
		for range 8 {
			// El desplazamiento a la derecha implementa el polinomio reflejado.
			if crc&1 != 0 {
				crc = (crc >> 1) ^ CRC32ReflectedPolynomial
			} else {
				crc >>= 1
			}
		}
	}
	return crc ^ 0xFFFFFFFF, nil
}

// CRC32ChecksumBits serializa el resultado MSB primero aunque el calculo sea reflejado.
func CRC32ChecksumBits(bits string) (string, error) {
	crc, err := CRC32Calculate(bits)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%032b", crc), nil
}

// CRC32Encode conserva los datos y agrega 32 bits de checksum al final.
func CRC32Encode(dataBits string) (string, error) {
	checksum, err := CRC32ChecksumBits(dataBits)
	if err != nil {
		return "", err
	}
	return dataBits + checksum, nil
}

// CRC32Verify separa los datos con messageLength y compara el checksum recibido.
func CRC32Verify(frameBits string, messageLength int) (bool, string, string) {
	if err := validateBits(frameBits); err != nil {
		return false, "", err.Error()
	}
	if messageLength < 0 || len(frameBits) != messageLength+32 {
		return false, "", "longitud CRC-32 invalida"
	}
	data := frameBits[:messageLength]
	received := frameBits[messageLength:]
	expected, _ := CRC32ChecksumBits(data)
	if received != expected {
		return false, "", fmt.Sprintf("CRC-32 no coincide: recibido %s, esperado %s", received, expected)
	}
	return true, data, ""
}
