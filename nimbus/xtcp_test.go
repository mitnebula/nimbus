package main

import (
	"testing"
	"time"
)

func TestSetXtcpCwnd(t *testing.T) {
	rtt := time.Duration(1) * time.Second
	fr := 120e6

	testXtcpData := &xtcpDataContainer{
		numVirtualFlows: 100,
		currVirtFlow:    0,
		virtual_cwnds:   make(map[uint16]float64),
		seq_nos:         make(map[uint16]uint32),
		recv_seq_nos:    make(map[uint16]uint32),
		drop_time:       make(map[uint16]int64),
	}

	testXtcpData.setXtcpCwnd(fr, rtt)

	// should be 1 sec * 120e6 bps = 120e6 bits / 12000 * 100 = 100

	for vfid := uint16(0); vfid < testXtcpData.numVirtualFlows; vfid++ {
		if cwnd := testXtcpData.virtual_cwnds[vfid]; cwnd != 100 {
			t.Errorf("wrong cwnd: %v should be 100", cwnd)
		}
	}
}

func BenchmarkGetNextSeq(b *testing.B) {
	testXtcpData := &xtcpDataContainer{
		numVirtualFlows: 100,
		currVirtFlow:    0,
		virtual_cwnds:   make(map[uint16]float64),
		seq_nos:         make(map[uint16]uint32),
		recv_seq_nos:    make(map[uint16]uint32),
		drop_time:       make(map[uint16]int64),
	}

	rtt := time.Duration(1) * time.Second
	fr := 120e6
	testXtcpData.setXtcpCwnd(fr, rtt)

	for i := 0; i < b.N; i++ {
		_, _ = testXtcpData.getNextSeq()
	}
}
