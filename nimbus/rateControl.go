package main

import (
	//"fmt"
	"math"
	"time"
)

const (
	est_bandwidth = 120e6

	alpha = 1
)

var beta float64

func shouldSwitch(zt float64, rtt time.Duration) {
	return // TODO switching
}

func updateRateDelay(
	rt float64,
	est_bandwidth float64,
	rin float64,
	zt float64,
	rtt time.Duration,
) float64 {
	beta = (rin / rtt.Seconds()) * 0.33
	newRate := rin + alpha*(est_bandwidth-zt-rin) - (beta/2)*(rtt.Seconds()-(1.25*min_rtt.Seconds()))

	minRate := 1500 * 8.0 / min_rtt.Seconds() // send at least 1 packet per rtt
	if newRate < minRate || math.IsNaN(newRate) {
		newRate = minRate
	}

	//fmt.Printf(" alpha_term: %.3v beta_term: %.3v rate: %.3v -> %.3v\n", alpha*(est_bandwidth-zt-rin), beta*(rtt.Seconds()-(1.1*min_rtt.Seconds())), rt, newRate)
	return newRate
}

func flowRateUpdater() {
	for {
		var wait time.Duration
		// update rate every ~rtt
		lv, err := rtts.Latest()
		if err != nil {
			wait = time.Duration(5) * time.Millisecond
		} else {
			wait = time.Duration(lv.(durationLogVal)) / 3
		}
		<-time.After(wait)

		lv, err = rtts.Latest()
		if err != nil {
			continue
		}
		rtt := time.Duration(lv.(durationLogVal))

		rout, oldPkt, newPkt, err := ThroughputFromTimes(ackTimes, time.Now(), rtt)
		if err != nil {
			continue
		}

		rin, err := ThroughputFromPackets(sendTimes, oldPkt, newPkt)
		if err != nil {
			continue
		}

		rin_history.Add(floatLogVal(rin))

		zt := est_bandwidth*(rin/rout) - rin

		//fmt.Printf("time: %v rtt: %v/%v rin: %.3v rout: %.3v zt: %.3v", Now(), rtt, min_rtt, rin, rout, zt)

		//shouldSwitch(zt, rtt)

		flowRateLock.Lock()

		switch flowMode {
		case DELAY:
			flowRate = updateRateDelay(flowRate, est_bandwidth, rin, zt, rtt)
		case XTCP:
			flowRate = xtcpData.updateRateXtcp(rtt)
			panic(false)
		}

		if flowRate < 0 {
			panic("negative flow rate")
		}

		flowRateLock.Unlock()
	}
}
