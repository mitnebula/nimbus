package main

import (
	"fmt"
	"net"
	"reflect"
	"syscall"
)

const SO_MAX_PACING_RATE = 47

func SetPacingRate(conn *net.UDPConn, rate float64) error {
	sockfd, err := SocketOf(conn)
	if err != nil {
		return err
	}

	err = syscall.SetsockoptInt(sockfd, syscall.SOL_SOCKET, SO_MAX_PACING_RATE, int(rate))
	if err != nil {
		return err
	}

	return nil
}

func SocketOf(c net.Conn) (int, error) {
	// re-implement go internal package: https://godoc.org/golang.org/x/net/internal/netreflect
	// implemented: golang/net/internal/netreflect/socket_posix.go:socketOf()

	v := reflect.ValueOf(c)
	switch e := v.Elem(); e.Kind() {
	case reflect.Struct:
		fd := e.FieldByName("conn").FieldByName("fd")
		switch e := fd.Elem(); e.Kind() {
		case reflect.Struct:
			sysfd := e.FieldByName("sysfd")
			return int(sysfd.Int()), nil
		}
	}
	return 0, fmt.Errorf("invalid type")
}
