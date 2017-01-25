package main

import (
	"bytes"
	"encoding/binary"
)

// encode the nimbus fields and pad the returned value to the given size
// if size < 16 (nimbus header size), buffer will not be padded
func encode(p Packet, size int) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p.Echo)
	if err != nil {
		return buf.Bytes(), err
	}

	err = binary.Write(buf, binary.LittleEndian, p.RecvTime)
	if err != nil {
		return buf.Bytes(), err
	}

	// size of encoded values above is 16 bytes
	if size > 16 {
		pad(buf, size)
	}

	return buf.Bytes(), err
}

func encodeInt64(t int64, buf []byte) error {
	b := bytes.NewBuffer(buf)
	b.Reset()
	err := binary.Write(b, binary.LittleEndian, t)
	return err
}

func pad(buf *bytes.Buffer, size int) {
	// header 22 bytes
	payload := bytes.Repeat([]byte("p"), size-16)
	buf.Write(payload)
}

func decode(b []byte) (Packet, error) {
	var p Packet
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &p.Echo)
	if err != nil {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &p.RecvTime)
	if err != nil {
		return p, err
	}

	return p, err
}
