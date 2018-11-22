package network

import (
	"encoding/binary"
	"io"
	"math"
)

func ReadUint8(reader io.Reader) (uint8, error) {
	buf := make([]byte, 1)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return 0, err
	}

	return buf[0], nil
}

func WriteUint8(writer io.Writer, value uint8) error {
	buf := make([]byte, 1)
	buf[0] = value

	_, err := writer.Write(buf)

	return err
}

func ReadUint16(reader io.Reader) (uint16, error) {
	buf := make([]byte, 2)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return 0, err
	}

	value := binary.BigEndian.Uint16(buf)

	return value, nil
}

func WriteUint16(writer io.Writer, value uint16) error {
	buf := make([]byte, 2)

	binary.BigEndian.PutUint16(buf, value)

	_, err := writer.Write(buf)

	return err
}

func ReadUint32(reader io.Reader) (uint32, error) {
	buf := make([]byte, 4)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return 0, err
	}

	value := binary.BigEndian.Uint32(buf)

	return value, nil
}

func WriteUint32(writer io.Writer, value uint32) error {
	buf := make([]byte, 4)

	binary.BigEndian.PutUint32(buf, value)

	_, err := writer.Write(buf)

	return err
}

func ReadUint64(reader io.Reader) (uint64, error) {
	buf := make([]byte, 8)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return 0, err
	}

	value := binary.BigEndian.Uint64(buf)

	return value, nil
}

func WriteUint64(writer io.Writer, value uint64) error {
	buf := make([]byte, 8)

	binary.BigEndian.PutUint64(buf, value)

	_, err := writer.Write(buf)

	return err
}

func ReadVarUint(reader io.Reader) (uint64, error) {
	size, err := ReadUint8(reader)
	if err != nil {
		return 0, err
	}

	switch size {
	case 0xff:
		return ReadUint64(reader)

	case 0xfe:
		value, err := ReadUint32(reader)
		if err != nil {
			return 0, err
		}

		return uint64(value), nil

	case 0xfd:
		value, err := ReadUint16(reader)
		if err != nil {
			return 0, err
		}

		return uint64(value), nil

	default:
		return uint64(size), nil
	}
}

func WriteVarUint(writer io.Writer, value uint64) error {
	if value < 0xfd {
		return WriteUint8(writer, uint8(value))
	}

	if value <= math.MaxUint16 {
		err := WriteUint8(writer, 0xfd)
		if err != nil {
			return err
		}

		return WriteUint16(writer, uint16(value))
	}

	if value <= math.MaxUint32 {
		err := WriteUint8(writer, 0xfe)
		if err != nil {
			return err
		}

		return WriteUint32(writer, uint32(value))
	}

	err := WriteUint8(writer, 0xff)
	if err != nil {
		return err
	}

	return WriteUint64(writer, value)
}

func ReadString(reader io.Reader, size uint64) (string, error) {
	buf := make([]byte, size)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func WriteString(writer io.Writer, value string) error {
	return nil
}
