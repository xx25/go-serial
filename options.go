//
// Copyright 2019-2022 Veniamin Albaev <albenik@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

type Option func(p *Port)

func WithBaudrate(o int) Option {
	return func(p *Port) {
		p.baudRate = o
	}
}

func WithDataBits(o int) Option {
	return func(p *Port) {
		p.dataBits = o
	}
}

func WithParity(o Parity) Option {
	return func(p *Port) {
		p.parity = o
	}
}

func WithStopBits(o StopBits) Option {
	return func(p *Port) {
		p.stopBits = o
	}
}

func WithReadTimeout(o int) Option {
	return func(p *Port) {
		p.setReadTimeoutValues(o)
	}
}

func WithWriteTimeout(o int) Option {
	return func(p *Port) {
		p.setWriteTimeoutValues(o)
	}
}

func WithHUPCL(o bool) Option {
	return func(p *Port) {
		p.hupcl = o
	}
}

// WithCtsRts enables (o=true) or disables (o=false) hardware RTS/CTS
// flow control on the port. The default is false (no flow control),
// preserving the historical behaviour of this library. When enabled,
// reconfigure() sets the platform flow-control flags (CRTSCTS on
// termios platforms; RTS-handshake + OutxCtsFlow on Windows) so the
// kernel pauses transmission when the DCE deasserts CTS and asserts
// RTS to throttle the peer — required for reliable bulk transfers over
// a real modem whose DTE/DCE rates differ.
func WithCtsRts(o bool) Option {
	return func(p *Port) {
		p.ctsRts = o
	}
}
