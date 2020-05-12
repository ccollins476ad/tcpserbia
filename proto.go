package main

import (
	"encoding/base64"
	"net"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"
)

var nc net.Conn

func txErr(c *SerConn, text string) error {
	return c.txCmd("error " + text)
}

func txAck(c *SerConn) error {
	return c.txCmd("ack")
}

func txRx(c *SerConn, val []byte) error {
	b64 := base64.StdEncoding.EncodeToString(val)
	return c.txCmd("rx " + b64)
}

func processConnect(c *SerConn, addrString string) {
	log.Infof("connecting to %v\n", addrString)

	if nc != nil {
		txErr(c, "already connected")
		return
	}

	var err error
	nc, err = net.Dial("tcp", addrString)
	if err != nil {
		txErr(c, err.Error())
		return
	}

	log.Infof("connected to %v\n", addrString)
	txAck(c)

	go func() {
		buf := make([]byte, 180)
		for {
			n, err := nc.Read(buf)
			if err != nil {
				nc = nil
				break
			}

			txRx(c, buf[:n])
		}
	}()
}

func processDisconnect(c *SerConn) {
	if nc == nil {
		txErr(c, "already disconnected")
		return
	}

	nc.Close()
	nc = nil

	txAck(c)
}

func processTx(c *SerConn, b64 string) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		txErr(c, err.Error())
		return
	}

	if nc == nil {
		txErr(c, "not connected")
		return
	}

	log.Debugf("net transmit: %s", string(data))
	for len(data) > 0 {
		n, err := nc.Write(data)
		if err != nil {
			txErr(c, err.Error())
			return
		}

		data = data[n:]
	}

	txAck(c)
}

func processCmd(c *SerConn, cmd []byte) {
	fields := strings.Fields(string(cmd))
	switch fields[0] {
	case "connect":
		if len(fields) < 2 {
			txErr(c, "expected addr-string")
			return
		}

		processConnect(c, fields[1])

	case "tx":
		if len(fields) < 2 {
			txErr(c, "expected byte-string")
			return
		}

		processTx(c, fields[1])

	case "disconnect":
		processDisconnect(c)
	}
}

func processCmds(c *SerConn) error {
	for {
		select {
		case buf := <-c.RxCh:
			processCmd(c, buf)

		case err := <-c.ErrCh:
			return err
		}
	}
}
