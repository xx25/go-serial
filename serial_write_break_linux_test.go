//
// Copyright 2026 FidoMail contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build linux && !android

package serial

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestWriteBreaksOnClose is the acceptance gate for the breakable-write
// change. A peer that stops draining is the faithful proxy for a modem
// that has deasserted CTS under hardware flow control: the pty output
// buffer fills and the kernel will accept no more bytes. The pre-change
// implementation did a blocking unix.Write FIRST and only consulted the
// close-pipe afterwards, so a Close() could not break a parked write —
// this test would hang. With select-before-write on a non-blocking fd,
// Close() wakes the writer's select and Write returns PortClosed promptly.
func TestWriteBreaksOnClose(t *testing.T) {
	slave, master := openPTYSlave(t)
	defer unix.Close(master)

	p, err := Open(slave)
	if err != nil {
		t.Fatalf("open %s: %v", slave, err)
	}
	// Intentionally never read from master: the slave's output buffer
	// fills and further writes cannot complete.

	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	go func() {
		// Far larger than any pty buffer, so the write cannot drain
		// without a reader and the writer parks in select.
		n, werr := p.Write(make([]byte, 1<<20))
		done <- result{n, werr}
	}()

	// Let the writer fill the buffer and block in select.
	time.Sleep(200 * time.Millisecond)
	select {
	case r := <-done:
		t.Fatalf("Write returned before Close (n=%d err=%v); the full buffer should have blocked it", r.n, r.err)
	default:
	}

	// Break it.
	if cerr := p.Close(); cerr != nil {
		t.Fatalf("Close: %v", cerr)
	}

	select {
	case r := <-done:
		var pe *PortError
		if !errors.As(r.err, &pe) || pe.Code() != PortClosed {
			t.Fatalf("Write after Close: got err=%v, want PortClosed", r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not return within 2s of Close — still blocked (the M2a regression)")
	}
}

// TestWriteCompletesWhenDrained confirms the rewritten Write still
// delivers every byte on a healthy link: a reader draining the master
// lets a large write finish with n == len and no error, exercising the
// non-blocking partial-write loop.
func TestWriteCompletesWhenDrained(t *testing.T) {
	slave, master := openPTYSlave(t)
	defer unix.Close(master)

	p, err := Open(slave)
	if err != nil {
		t.Fatalf("open %s: %v", slave, err)
	}
	defer p.Close()

	const total = 256 * 1024
	// Drain the master continuously so the slave's buffer never wedges.
	go func() {
		buf := make([]byte, 4096)
		read := 0
		for read < total {
			n, rerr := unix.Read(master, buf)
			if n > 0 {
				read += n
			}
			if rerr != nil && !errors.Is(rerr, unix.EINTR) && !errors.Is(rerr, unix.EAGAIN) {
				return
			}
		}
	}()

	n, werr := p.Write(make([]byte, total))
	if werr != nil {
		t.Fatalf("Write: %v", werr)
	}
	if n != total {
		t.Fatalf("Write short: got %d, want %d", n, total)
	}
}
