package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"
)

var nc net.Conn
var nl net.Listener

func txErr(c *SerConn, format string, args ...interface{}) error {
	return c.txCmd("error " + fmt.Sprintf(format, args...))
}

func txAck(c *SerConn) error {
	return c.txCmd("ack")
}

func txRx(c *SerConn, val []byte) error {
	b64 := base64.StdEncoding.EncodeToString(val)
	return c.txCmd("rx " + b64)
}

func txAccept(c *SerConn) error {
	return c.txCmd(fmt.Sprintf("accept %v", nc.RemoteAddr()))
}

func txClose(c *SerConn, reason string) error {
	return c.txCmd("close " + reason)
}

func connLoop(c *SerConn) {
	buf := make([]byte, 180)
	for {
		n, err := nc.Read(buf)
		if err != nil {
			txClose(c, err.Error())
			nc = nil
			break
		}

		txRx(c, buf[:n])
	}
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

	go connLoop(c)
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

func processListen(c *SerConn, portString string) {
	port, err := strconv.ParseInt(portString, 0, 16)
	if err != nil {
		txErr(c, "invalid port: %v", err)
		return
	}

	if nl != nil {
		txErr(c, "already listening")
		return
	}

	if nc != nil {
		txErr(c, "connection already in use")
		return
	}

	nl, err = net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		txErr(c, "%v", err)
		return
	}

	go func() {
		var err error

		nc, err = nl.Accept()
		if err != nil {
			nl = nil
			return
		}

		go connLoop(c)

		txAccept(c)
	}()

	txAck(c)
}

func processStopListen(c *SerConn) {
	if nl == nil {
		txErr(c, "not listening")
		return
	}

	nl.Close()
	nl = nil

	txAck(c)
}

func processReset(c *SerConn) {
	if nc != nil {
		nc.Close()
		nc = nil
	}

	if nl != nil {
		nl.Close()
		nl = nil
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

	case "listen":
		if len(fields) < 2 {
			txErr(c, "expected port-string")
			return
		}
		processListen(c, fields[1])

	case "stop-listen":
		processStopListen(c)

	case "reset":
		processReset(c)
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
