//
// Copyright 2026 FidoMail contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

//go:build linux && !android

package serial

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"
)

// openPTYSlave allocates a fresh pseudo-terminal pair via /dev/ptmx and
// returns the slave device path (e.g. /dev/pts/7) together with the
// still-open master fd. The caller must close the master to release the
// pair. A pts honours the CRTSCTS termios bit (it is stored and read
// back even though the pty layer has no real RTS/CTS lines), which is
// exactly what lets this test assert the flow-control plumbing without
// physical hardware.
func openPTYSlave(t *testing.T) (slavePath string, master int) {
	t.Helper()
	m, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("open /dev/ptmx: %v", err)
	}
	n, err := unix.IoctlGetInt(m, unix.TIOCGPTN)
	if err != nil {
		_ = unix.Close(m)
		t.Fatalf("TIOCGPTN: %v", err)
	}
	// grantpt + unlockpt equivalent: clear the slave lock.
	if err := unix.IoctlSetPointerInt(m, unix.TIOCSPTLCK, 0); err != nil {
		_ = unix.Close(m)
		t.Fatalf("TIOCSPTLCK: %v", err)
	}
	return fmt.Sprintf("/dev/pts/%d", n), m
}

// ctsRtsEnabled reopens the slave's termios and reports whether the
// CRTSCTS bit survived the last reconfigure().
func ctsRtsEnabled(t *testing.T, p *Port) bool {
	t.Helper()
	s, err := p.retrieveTermSettings()
	if err != nil {
		t.Fatalf("retrieveTermSettings: %v", err)
	}
	return s.termios.Cflag&tcCRTSCTS != 0
}

// TestWithCtsRts pins the new flow-control knob: WithCtsRts(true) leaves
// CRTSCTS set on the port termios, while the default (and an explicit
// WithCtsRts(false)) leaves it clear — the historical behaviour.
func TestWithCtsRts(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		slave, master := openPTYSlave(t)
		defer unix.Close(master)

		p, err := Open(slave, WithCtsRts(true))
		if err != nil {
			t.Fatalf("open %s with WithCtsRts(true): %v", slave, err)
		}
		defer p.Close()

		if !ctsRtsEnabled(t, p) {
			t.Fatalf("CRTSCTS not set after WithCtsRts(true)")
		}
	})

	t.Run("default_off", func(t *testing.T) {
		slave, master := openPTYSlave(t)
		defer unix.Close(master)

		p, err := Open(slave)
		if err != nil {
			t.Fatalf("open %s: %v", slave, err)
		}
		defer p.Close()

		if ctsRtsEnabled(t, p) {
			t.Fatalf("CRTSCTS set with no option (expected default off)")
		}
	})

	t.Run("explicit_off", func(t *testing.T) {
		slave, master := openPTYSlave(t)
		defer unix.Close(master)

		p, err := Open(slave, WithCtsRts(false))
		if err != nil {
			t.Fatalf("open %s with WithCtsRts(false): %v", slave, err)
		}
		defer p.Close()

		if ctsRtsEnabled(t, p) {
			t.Fatalf("CRTSCTS set after WithCtsRts(false)")
		}
	})

	t.Run("reconfigure_toggles", func(t *testing.T) {
		slave, master := openPTYSlave(t)
		defer unix.Close(master)

		p, err := Open(slave)
		if err != nil {
			t.Fatalf("open %s: %v", slave, err)
		}
		defer p.Close()

		if err := p.Reconfigure(WithCtsRts(true)); err != nil {
			t.Fatalf("reconfigure on: %v", err)
		}
		if !ctsRtsEnabled(t, p) {
			t.Fatalf("CRTSCTS not set after Reconfigure(WithCtsRts(true))")
		}
		if err := p.Reconfigure(WithCtsRts(false)); err != nil {
			t.Fatalf("reconfigure off: %v", err)
		}
		if ctsRtsEnabled(t, p) {
			t.Fatalf("CRTSCTS still set after Reconfigure(WithCtsRts(false))")
		}
	})
}
