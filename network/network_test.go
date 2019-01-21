package network

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMessageHeader(t *testing.T) {
	messageHeader := &messageHeader{
		magic:   424140,
		msgType: "olala",
		length:  42,
	}

	buf := bytes.NewBuffer(nil)

	err := writeMessageHeader(buf, messageHeader)
	if err != nil {
		t.Errorf("Excepted no error, but got %s instead.", err)
	}

	messageHeaderAfter, err := readMessageHeader(buf)
	if err != nil {
		t.Errorf("Excepted no error, but got %s instead.", err)
	}

	if !reflect.DeepEqual(messageHeader, messageHeaderAfter) {
		t.Errorf("Excepted %v, but got %v instead.", messageHeader, messageHeaderAfter)
	}
}
