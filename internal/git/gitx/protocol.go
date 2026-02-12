package gitx

import (
	"fmt"
	"io"
)

// PackLine writes a pkt-line formatted string.
func PackLine(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%04x%s", len(s)+4, s)
	return err
}

// PackFlush writes a flush packet.
func PackFlush(w io.Writer) error {
	_, err := fmt.Fprint(w, "0000")
	return err
}

// PackError writes an ERR packet for protocol-level errors.
// Git displays this as: fatal: remote error: <msg>
func PackError(w io.Writer, msg string) error {
	return PackLine(w, "ERR "+msg)
}
