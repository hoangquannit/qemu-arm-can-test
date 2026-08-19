package socketcan

import "context"

// Bus is the small transport contract shared by the CLI, ECU process, and
// tests. It describes current behavior rather than a future provider system.
type Bus interface {
	Send(Frame) error
	Receive(context.Context) (Frame, error)
	Close() error
}
