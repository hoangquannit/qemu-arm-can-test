//go:build !linux

package socketcan

import "context"

type Conn struct{}

func Open(string) (*Conn, error) {
	return nil, ErrUnsupported
}

func (c *Conn) Send(Frame) error {
	return ErrUnsupported
}

func (c *Conn) Receive(context.Context) (Frame, error) {
	return Frame{}, ErrUnsupported
}

func (c *Conn) Close() error {
	return nil
}
