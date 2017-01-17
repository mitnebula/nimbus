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

var lastPrint time.Time

func init() {
	lastPrint = time.Now()
	setcwndcounter = 0
	xtcpData = &xtcpDataContainer{
		numVirtualFlows: 1,
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

	res := fr * ONE_PACKET / rtt.Seconds()
	if time.Since(lastPrint) > time.Duration(40)*time.Millisecond {
		fmt.Printf("time: %v xtcp_vcwnd: %v xtcp_curr_rate: %v rtt: %v\n", time.Since(startTime).Seconds(), fr, res, rtt)
		lastPrint = time.Now()
	}
	return res
}

func (xt *xtcpDataContainer) getNextSeq() (seq uint32, vfid uint16) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	seq, vfid = xt.seq_nos[xt.currVirtFlow], xt.currVirtFlow
	xt.seq_nos[vfid]++

	nextFlow := (vfid + 1) % xt.numVirtualFlows
	//xt.currVirtFlow = nextFlow
	if xt.seq_nos[vfid] > xt.seq_nos[nextFlow]+180 {
		xt.currVirtFlow = nextFlow
	}

	return
}

func (xt *xtcpDataContainer) setXtcpCwnd(flowRate float64) {
	setcwndcounter++ // TODO remove this sanity check (for no competition case)
	if setcwndcounter > 1 {
		panic(false)
	}
	for vfid := uint16(0); vfid < xt.numVirtualFlows; vfid++ {
		// TODO use curr rtt instead of 0.165
		xt.virtual_cwnds[vfid] = (0.165 * flowRate) / float64(8*1500*xt.numVirtualFlows)
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
		if Now() > xt.drop_time[vfid] {
			fmt.Println("drop", vfid, xt.virtual_cwnds[vfid])
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
}

// assume lock already acquired
func (xt *xtcpDataContainer) switchToXtcp(flowRate float64) {
	fmt.Println("switching to xtcp")
	flowMode = XTCP
	xt.setXtcpCwnd(flowRate)
}

func (xt *xtcpDataContainer) checkXtcpSeq(fid uint16, seq uint32) bool {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	expected := xt.recv_seq_nos[fid]
	if seq < expected {
		err := fmt.Errorf("seq out of order: %v %v fid %v", seq, expected, fid)
		panic(err)
	}

	xt.recv_seq_nos[fid] = seq + 1
	return seq == expected
}

func (xt *xtcpDataContainer) increaseXtcpWind(fid uint16) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	denom := xt.virtual_cwnds[fid] * float64(xt.numVirtualFlows)

	for f, _ := range xt.virtual_cwnds {
		xt.virtual_cwnds[f] += 1.0 / denom
	}
}
