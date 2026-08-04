package algorithms

import "testing"

func asciiBits(value string) string {
	result := ""
	for _, char := range []byte(value) {
		for shift := 7; shift >= 0; shift-- {
			if char&(1<<shift) != 0 {
				result += "1"
			} else {
				result += "0"
			}
		}
	}
	return result
}

func TestCRC32IEEEKnownVector(t *testing.T) {
	crc, err := CRC32Calculate(asciiBits("123456789"))
	if err != nil {
		t.Fatal(err)
	}
	if crc != 0xCBF43926 {
		t.Fatalf("CRC = %08X, se esperaba CBF43926", crc)
	}
}

func TestCRC32ShortPadding(t *testing.T) {
	crc, err := CRC32Calculate("1")
	if err != nil {
		t.Fatal(err)
	}
	if crc != 0xCC1D6927 {
		t.Fatalf("CRC = %08X, se esperaba CC1D6927", crc)
	}
}

func TestCRC32DetectsErrors(t *testing.T) {
	data := asciiBits("ATM")
	frame, _ := CRC32Encode(data)
	for _, index := range []int{0, len(data) - 1, len(frame) - 1} {
		corrupted := []byte(frame)
		if corrupted[index] == '0' {
			corrupted[index] = '1'
		} else {
			corrupted[index] = '0'
		}
		ok, _, _ := CRC32Verify(string(corrupted), len(data))
		if ok {
			t.Fatalf("no se detecto el error en %d", index)
		}
	}
}
