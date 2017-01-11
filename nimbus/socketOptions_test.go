package main

import (
	"fmt"
	"net"
	"testing"
)

func BenchmarkSocketSyscall(b *testing.B) {
	addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:42425")
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < b.N; i++ {
		SetPacingRate(conn, 48e6)
	}
}
