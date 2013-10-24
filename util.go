package sgip

import (
	"bytes"
	"fmt"
	"strconv"
)

func logBytes(data []byte, prefix string) {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	for _, b := range data {
		buf.WriteString(fmt.Sprintf(" %02X", b))
	}
	sgipConfig.Logger.Info(buf.String())
}

func bytesToIntBig(bytes []byte) int {
	res := 0
	for i := 0; i < len(bytes); i++ {
		res = (res << 8) | int(bytes[i])
	}
	return res
}

func intToBytesBig(i int, b []byte) {
	b[0] = byte(i >> 24)
	b[1] = byte(i >> 16)
	b[2] = byte(i >> 8)
	b[3] = byte(i)
}

func decodeBytesString(bytes []byte) string {
	var i int
	for i = 0; i < len(bytes); i++ {
		if bytes[i] == 0 {
			break
		}
	}

	return string(bytes[0:i])
}

func encodeStringBytes(s string, bytes []byte) {
	for i := 0; i < len(bytes); i++ {
		if i < len(s) {
			bytes[i] = byte(s[i])
		} else {
			bytes[i] = 0
		}
	}
}

func bytesToHexString(in []byte) string {
	var buf bytes.Buffer

	for i := 0; i < len(in); i++ {
		buf.WriteString(fmt.Sprintf("%02X", in[i]))
	}

	return buf.String()
}

func hexStringToBytes(s string) ([]byte, error) {
	count := len(s) / 2
	res := make([]byte, count)
	for i := 0; i < count; i++ {
		v, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, err
		}

		res[i] = byte(v)
	}

	return res, nil
}
