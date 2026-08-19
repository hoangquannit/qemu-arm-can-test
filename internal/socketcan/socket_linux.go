//go:build linux

package socketcan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const canFrameSize = 16

type Conn struct {
	fd int
}

func Open(interfaceName string) (*Conn, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("find CAN interface %q: %w", interfaceName, err)
	}

	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.CAN_RAW)
	if err != nil {
		return nil, fmt.Errorf("open SocketCAN socket: %w", err)
	}

	if err := unix.Bind(fd, &unix.SockaddrCAN{Ifindex: iface.Index}); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("bind SocketCAN socket to %q: %w", interfaceName, err)
	}

	return &Conn{fd: fd}, nil
}

func (c *Conn) Send(frame Frame) error {
	if err := frame.Validate(); err != nil {
		return err
	}

	buffer := make([]byte, canFrameSize)
	*(*uint32)(unsafe.Pointer(&buffer[0])) = frame.ID
	buffer[4] = byte(len(frame.Data))
	copy(buffer[8:], frame.Data)

	written, err := unix.Write(c.fd, buffer)
	if err != nil {
		return fmt.Errorf("send CAN frame %s: %w", frame, err)
	}
	if written != canFrameSize {
		return fmt.Errorf("send CAN frame %s: wrote %d of %d bytes", frame, written, canFrameSize)
	}
	return nil
}

func (c *Conn) Receive(ctx context.Context) (Frame, error) {
	for {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return Frame{}, ErrTimeout
			}
			return Frame{}, err
		}

		wait := 100 * time.Millisecond
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return Frame{}, ErrTimeout
			}
			if remaining < wait {
				wait = remaining
			}
		}

		pollFDs := []unix.PollFd{{Fd: int32(c.fd), Events: unix.POLLIN}}
		ready, err := unix.Poll(pollFDs, int(wait.Milliseconds()))
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return Frame{}, fmt.Errorf("poll SocketCAN socket: %w", err)
		}
		if ready == 0 {
			continue
		}

		buffer := make([]byte, canFrameSize)
		read, err := unix.Read(c.fd, buffer)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return Frame{}, fmt.Errorf("receive CAN frame: %w", err)
		}
		if read != canFrameSize {
			return Frame{}, fmt.Errorf("receive CAN frame: read %d of %d bytes", read, canFrameSize)
		}

		id := *(*uint32)(unsafe.Pointer(&buffer[0])) & unix.CAN_SFF_MASK
		length := int(buffer[4])
		if length > MaxDataLength {
			return Frame{}, fmt.Errorf("receive CAN frame: invalid payload length %d", length)
		}
		data := make([]byte, length)
		copy(data, buffer[8:8+length])
		return Frame{ID: id, Data: data}, nil
	}
}

func (c *Conn) Close() error {
	if c.fd < 0 {
		return nil
	}
	err := unix.Close(c.fd)
	c.fd = -1
	return err
}
