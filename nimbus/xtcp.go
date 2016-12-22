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

var xtcpData xtcpDataContainer

func init() {
	xtcpData = xtcpDataContainer{
		xtcp_mode:       false,
		numVirtualFlows: 10,
		currVirtFlow:    0,
		virtual_cwnds:   make(map[int]float64),
		seq_nos:         make(map[int]int),
		recv_seq_nos:    make(map[int]int),
		drop_time:       make(map[int]int64),
	}

	setXtcpCwnd(flowRate)
	for vfid := 0; vfid < xtcpData.numVirtualFlows; vfid++ {
		xtcpData.seq_nos[vfid] = 0
		xtcpData.recv_seq_nos[vfid] = 0
		xtcpData.drop_time[vfid] = 0
	}
}

func updateRateXtcp(
	rtt time.Duration,
) float64 {
	flowRate := 0.0
	xtcpData.mut.Lock()
	defer xtcpData.mut.Unlock()

	fmt.Println(xtcpData.virtual_cwnds)
	for i := range xtcpData.virtual_cwnds {
		flowRate += xtcpData.virtual_cwnds[i]
	}
	fmt.Printf("curr rate: %f curr_rtt: %f\n", flowRate, rtt.Seconds())
	return flowRate * (1480 * 8.0) / rtt.Seconds()
}

func setXtcpCwnd(flowRate float64) {
	for vfid := 0; vfid < xtcpData.numVirtualFlows; vfid++ {
		xtcpData.virtual_cwnds[vfid] = (0.165 * flowRate) / float64(8*1480*xtcpData.numVirtualFlows)
	}
	//fmt.Println("set xtcp cwnds to ", xtcpData.virtual_cwnds)
}

func dropDetected(vfid int) {
	xtcpData.mut.Lock()
	defer xtcpData.mut.Unlock()

	if !xtcpData.xtcp_mode {
		flowRateLock.Lock()
		defer flowRateLock.Unlock()
		switchToXtcp(flowRate)
	} else if xtcpData.drop_time[vfid] <= Now() {
		xtcpData.virtual_cwnds[vfid] *= 0.5
		lv, err := rtts.Latest()
		if err != nil {
			return
		}

		xtcpData.drop_time[vfid] = Now() + time.Duration(lv.(LogDuration)).Nanoseconds()
	}
}

func switchToXtcp(flowRate float64) {
	xtcpData.xtcp_mode = true
	setXtcpCwnd(flowRate)
}

func incrementXtcpSeq() {
	xtcpData.mut.Lock()
	defer xtcpData.mut.Unlock()
	xtcpData.seq_nos[xtcpData.currVirtFlow]++
	xtcpData.currVirtFlow = (xtcpData.currVirtFlow + 1) % xtcpData.numVirtualFlows
}

func checkXtcpSeq(fid int, seq int) bool {
	xtcpData.mut.Lock()
	defer xtcpData.mut.Unlock()

	expected := xtcpData.recv_seq_nos[fid]
	if seq < expected {
		panic(false)
	}
	xtcpData.recv_seq_nos[fid]++
	return seq == expected
}

func increaseXtcpWind(fid int) {
	xtcpData.mut.Lock()
	defer xtcpData.mut.Unlock()

	xtcpData.virtual_cwnds[fid] += 1.0 / xtcpData.virtual_cwnds[fid]
}
