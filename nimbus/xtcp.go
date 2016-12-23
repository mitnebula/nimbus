package main

import (
	"fmt"
	"sync"
	"time"
)

type xtcpDataContainer struct {
	xtcp_mode       bool
	numVirtualFlows int
	currVirtFlow    int
	virtual_cwnds   map[int]float64
	seq_nos         map[int]int
	recv_seq_nos    map[int]int
	drop_time       map[int]int64
	mut             sync.Mutex
}

var xtcpData *xtcpDataContainer
var setcwndcounter int

func init() {
	setcwndcounter = 0
	xtcpData = &xtcpDataContainer{
		xtcp_mode:       false,
		numVirtualFlows: 10,
		currVirtFlow:    0,
		virtual_cwnds:   make(map[int]float64),
		seq_nos:         make(map[int]int),
		recv_seq_nos:    make(map[int]int),
		drop_time:       make(map[int]int64),
	}

	xtcpData.setXtcpCwnd(flowRate)
	for vfid := 0; vfid < xtcpData.numVirtualFlows; vfid++ {
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

func (xt *xtcpDataContainer) getNextSeq() (seq int, vfid int) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	seq, vfid = xt.seq_nos[xt.currVirtFlow], xt.currVirtFlow
	xt.seq_nos[xt.currVirtFlow]++
	xt.currVirtFlow = (xt.currVirtFlow + 1) % xt.numVirtualFlows

	return
}

func (xt *xtcpDataContainer) setXtcpCwnd(flowRate float64) {
	setcwndcounter++
	if setcwndcounter > 1 {
		panic(false)
	}
	for vfid := 0; vfid < xt.numVirtualFlows; vfid++ {
		xt.virtual_cwnds[vfid] = (0.165 * flowRate) / float64(8*1480*xt.numVirtualFlows)
	}
}

func (xt *xtcpDataContainer) dropDetected(vfid int) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	if !xt.xtcp_mode {
		flowRateLock.Lock()
		defer flowRateLock.Unlock()
		xt.switchToXtcp(flowRate)
	} else if xt.drop_time[vfid] <= Now() {
		xt.virtual_cwnds[vfid] *= 0.5
		if xt.virtual_cwnds[vfid] < 1 {
			xt.virtual_cwnds[vfid] = 1
		}
		lv, err := rtts.Latest()
		if err != nil {
			return
		}

		xt.drop_time[vfid] = Now() + time.Duration(lv.(LogDuration)).Nanoseconds()
	}
}

// assume lock already acquired
func (xt *xtcpDataContainer) switchToXtcp(flowRate float64) {
	fmt.Println("switching to xtcp")
	xt.xtcp_mode = true
	xt.setXtcpCwnd(flowRate)
}

func (xt *xtcpDataContainer) checkXtcpSeq(fid int, seq int) bool {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	expected := xt.recv_seq_nos[fid]
	if seq < expected {
		panic(false)
	}

	xt.recv_seq_nos[fid] = seq + 1
	return seq == expected
}

func (xt *xtcpDataContainer) increaseXtcpWind(fid int) {
	xt.mut.Lock()
	defer xt.mut.Unlock()

	xt.virtual_cwnds[fid] += 1.0 / xt.virtual_cwnds[fid]
}
