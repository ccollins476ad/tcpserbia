package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

type SerConn struct {
	RxCh  chan []byte
	ErrCh chan error
	port  *serial.Port
}

func NewSerConn(devPath string) (*SerConn, error) {
	cfg := &serial.Config{
		Name: devPath,
		Baud: 115200,
	}

	port, err := serial.OpenPort(cfg)
	if err != nil {
		return nil, err
	}

	err = port.Flush()
	if err != nil {
		return nil, err
	}

	c := &SerConn{
		RxCh:  make(chan []byte),
		ErrCh: make(chan error),
		port:  port,
	}

	go c.readLoop()

	return c, nil
}

func (c *SerConn) Close() error {
	close(c.ErrCh)
	return c.port.Close()
}

func (c *SerConn) readLoop() {
	var rsp []byte
	for {
		buf := make([]byte, 1)
		_, err := c.port.Read(buf)
		if err != nil {
			c.ErrCh <- err
			return
		}

		if buf[0] == '\n' {
			log.Debugf("serial receive: %s", string(rsp))
			c.RxCh <- rsp
			rsp = rsp[:0]
		} else {
			rsp = append(rsp, buf[0])
		}
	}
}

func (c *SerConn) txCmd(s string) error {
	log.Debugf("serial transmit: %s", s)

	buf := []byte(s + "\n")

	n, err := c.port.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("failed to transmit bytes over serial: only sent %d of %d bytes", n, len(buf))
	}

	return nil
}
