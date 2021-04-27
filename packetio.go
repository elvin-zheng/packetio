package packetio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type PacketIo interface {
	Read(ctx context.Context) (*Message, error)
	Write(message *Message) error
}

const (
	headerLen   = 4
	Version     = "1.0.0"
	MessageSign = "!@QESEFDSAID#$134"
)

func NewPacketIo(conn net.Conn) PacketIo {
	p := &Packetio{
		scan: bufio.NewScanner(conn),
		w:    bufio.NewWriter(conn),
	}
	p.scan.Split(p.split)
	return p
}

type Packetio struct {
	scan *bufio.Scanner
	w    *bufio.Writer
}

func (p *Packetio) Read(ctx context.Context) (*Message, error) {
	for p.scan.Scan() {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("read closed")
		default:
			err := p.scan.Err()
			if err != nil && err != io.EOF {
				return nil, err
			}

			bs := p.scan.Bytes()
			var msg = &Message{}
			if err := json.Unmarshal(bs, msg); err != nil {
				return nil, err
			}
			return msg, nil
		}
	}
	return nil, fmt.Errorf("read err")
}

func (p *Packetio) Write(m *Message) error {
	if bs, err := json.Marshal(m); err != nil {
		return err
	} else {
		var lenNum = make([]byte, headerLen)
		binary.BigEndian.PutUint32(lenNum, uint32(len(bs)))
		var buf = bytes.NewBuffer(lenNum)
		_, _ = buf.Write(bs)
		if _, err := p.w.Write(buf.Bytes()); err != nil {
			return err
		}
		return p.w.Flush()
	}
}

func (p *Packetio) split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) < 4 {
		return
	}
	length := binary.BigEndian.Uint32(data[:4])
	if !atEOF && length == uint32(len(data[4:])) {
		return len(data), data[4:], nil
	}
	return
}
