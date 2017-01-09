package main

import (
	"fmt"
	"sync"
	"time"
)

type xtcpDataContainer struct {
	numVirtualFlows uint16
	currVirtFlow    uint16
	virtual_cwnds   map[uint16]float64
	seq_nos         map[uint16]uint32
	recv_seq_nos    map[uint16]uint32
	drop_time       map[uint16]int64
	mut             sync.Mutex
}

var xtcpData *xtcpDataContainer
var setcwndcounter int

func init() {
	setcwndcounter = 0
	xtcpData = &xtcpDataContainer{
		numVirtualFlows: 10,
		currVirtFlow:    0,
		virtual_cwnds:   make(map[uint16]float64),
		seq_nos:         make(map[uint16]uint32),
		recv_seq_nos:    make(map[uint16]uint32),
		drop_time:       make(map[uint16]int64),
	}

	xtcpData.setXtcpCwnd(flowRate)
	for vfid := uint16(0); vfid < xtcpData.numVirtualFlows; vfid++ {
		xtcpData.seq_nos[vfid] = 0
		xtcpData.recv_seq_nos[vfid] = 0
		xtcpData.drop_time[vfid] = 0
	}
}

func (xt *xtcpDataContainer) updateRateXtcp(
	rtt time.Duration,
) float64 {
	fr := 0.0
	xt.mut.Lock()
	defer xt.mut.Unlock()

	for _, cwnd := range xt.virtual_cwnds {
		fr += cwnd
	}

	fr = fr * (1480 * 8.0) / rtt.Seconds()
	fmt.Printf("time: %v xtcp_curr_rate: %f curr_rtt: %v\n", Now(), fr, rtt)
	return fr
}

func (xt *xtcpDataContainer) getNextSeq() (seq uint32, vfid uint16) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	seq, vfid = xt.seq_nos[xt.currVirtFlow], xt.currVirtFlow
	xt.seq_nos[xt.currVirtFlow]++
	xt.currVirtFlow = (xt.currVirtFlow + 1) % xt.numVirtualFlows

	return
}

func (xt *xtcpDataContainer) setXtcpCwnd(flowRate float64) {
	setcwndcounter++ // TODO remove this sanity check (for no competition case)
	if setcwndcounter > 1 {
		panic(false)
	}
	for vfid := uint16(0); vfid < xt.numVirtualFlows; vfid++ {
		xt.virtual_cwnds[vfid] = (0.165 * flowRate) / float64(8*1480*xt.numVirtualFlows)
	}
}

func (xt *xtcpDataContainer) dropDetected(vfid uint16) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	switch flowMode {
	case BETAZERO:
		fallthrough
	case DELAY:
		flowRateLock.Lock()
		defer flowRateLock.Unlock()
		xt.switchToXtcp(flowRate)
	case XTCP:
		xt.virtual_cwnds[vfid] *= 0.5
		if xt.virtual_cwnds[vfid] < 1 {
			xt.virtual_cwnds[vfid] = 1
		}
		lv, err := rtts.Latest()
		if err != nil {
			return
		}

		xt.drop_time[vfid] = Now() + time.Duration(lv.(durationLogVal)).Nanoseconds()
	}
}

// assume lock already acquired
func (xt *xtcpDataContainer) switchToXtcp(flowRate float64) {
	fmt.Println("switching to xtcp")
	flowMode = XTCP
	xt.setXtcpCwnd(flowRate)
}

func (xt *xtcpDataContainer) checkXtcpSeq(fid uint16, seq uint32) (bool, uint32) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	expected := xt.recv_seq_nos[fid]
	if seq < expected {
		panic(false)
	}

	xt.recv_seq_nos[fid] = seq + 1
	return seq == expected, expected
}

func (xt *xtcpDataContainer) increaseXtcpWind(fid uint16) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	xt.virtual_cwnds[fid] += 1.0 / xt.virtual_cwnds[fid]
}
