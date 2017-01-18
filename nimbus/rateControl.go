package main

import (
	//"fmt"
	"math"
	"time"
)

const (
	est_bandwidth = 120e6

	alpha = 1

	// switching parameters
	delayToTestThresh = 0.05 // fraction of est_bw
	xtcpTimeout       = 10   // rtts
	testTimeout       = 6    // rtts
)

var beta float64

var zt_history *TimedLog
var switchTime time.Time

func init() {
	zt_history = InitTimedLog(min_rtt)
	switchTime = time.Now()

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33
}

func deltaZt(zt float64, rtt time.Duration) (float64, error) {
	oldZt, _, err := zt_history.Before(time.Now().Add(-1 * rtt))
	if err != nil {
		return 0, err
	}

	return zt - float64(oldZt.(floatLogVal)), nil
}

func switchFromDelayToTest() {
	return // TODO
}

func switchFromTestToDelay() {
	return // TODO
}

func switchFromXtcpToTest() {
	return // TODO
}

func switchFromTestToXtcp() {
	return // TODO
}

func shouldSwitch(zt float64, rtt time.Duration) {
	elapsed := time.Since(switchTime)
	if elapsed < 3*min_rtt {
		return
	}

	switch flowMode {
	case DELAY:
		// if delta zt > alpha * mu
		// go to test
		dZt, err := deltaZt(zt, rtt)
		if err != nil {
			return
		}

		if dZt > delayToTestThresh*est_bandwidth {
			switchFromDelayToTest()
		}
	case XTCP:
		// if timeout expires
		// go to test
		if elapsed > xtcpTimeout*min_rtt {
			switchFromXtcpToTest()
		}
	case TEST_FROM_DELAY:
		fallthrough
	case TEST_FROM_XTCP:
		// if timeout expires
		// go to delay
		if elapsed > testTimeout*min_rtt {
			switchFromTestToDelay()
		}

		// if after 3 rtts and delta zt > alpha * mu
		// go to xtcp
		dZt, err := deltaZt(zt, rtt)
		if err != nil {
			return
		}

		if dZt > delayToTestThresh*est_bandwidth {
			switchFromTestToXtcp()
		}
	}
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

func measure() (rin float64, rout float64, zt float64, avgRtt time.Duration, err error) {
	lv, err := rtts.Latest()
	if err != nil {
		return
	}
	rtt := time.Duration(lv.(durationLogVal))

	rout, oldPkt, newPkt, err := ThroughputFromTimes(ackTimes, time.Now(), rtt)
	if err != nil {
		return
	}

	rin, err = ThroughputFromPackets(sendTimes, oldPkt, newPkt)
	if err != nil {
		return
	}

	zt = est_bandwidth*(rin/rout) - rin

	//fmt.Printf("time: %v rtt: %v/%v rin: %.3v rout: %.3v zt: %.3v", Now(), rtt, min_rtt, rin, rout, zt)

	lv, err = rtts.Avg()
	if err != nil {
		avgRtt = rtt
	}
	avgRtt = time.Duration(lv.(durationLogVal))

	return
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

		doUpdate()
	}
}

func doUpdate() {
	rin, _, zt, rtt, err := measure()
	if err != nil {
		return
	}

	shouldSwitch(zt, rtt)
	zt_history.Add(time.Now(), floatLogVal(zt))

	flowRateLock.Lock()

	switch flowMode {
	case DELAY:
		flowRate = updateRateDelay(flowRate, est_bandwidth, rin, zt, rtt)
	case XTCP:
		flowRate = xtcpData.updateRateXtcp(rtt)
	}

	if flowRate < 0 {
		panic("negative flow rate")
	}

	flowRateLock.Unlock()

}
