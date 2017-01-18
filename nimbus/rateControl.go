package main

import (
	"fmt"
	"math"
	"time"
)

const (
	est_bandwidth = 48e6

	alpha = 1

	// switching parameters
	delayToTestThresh = 0.05 // fraction of est_bw
	xtcpTimeout       = 10   // rtts
	testTimeout       = 6    // rtts
)

var beta float64

var origFlowRate float64
var zt_history *TimedLog
var switchTime time.Time

var currMode string

func init() {
	zt_history = InitTimedLog(min_rtt)
	switchTime = time.Now()

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33
	origFlowRate = flowRate
}

func deltaZt(zt float64, rtt time.Duration) (float64, error) {
	oldZt, _, err := zt_history.Before(time.Now().Add(-1 * rtt))
	if err != nil {
		return 0, err
	}

	return zt - float64(oldZt.(floatLogVal)), nil
}

func switchFromDelayToTest() {
	fmt.Printf("%v : DELAY -> TEST\n", time.Now().UnixNano())
	flowMode = TEST_FROM_DELAY
	currMode = "TEST_FROM_DELAY"
	origFlowRate = flowRate
	switchTime = time.Now()
	return
}

func switchFromTestToDelay() {
	fmt.Printf("%v : TEST -> DELAY\n", time.Now().UnixNano())
	flowMode = DELAY
	currMode = "DELAY"
	switchTime = time.Now()
	return
}

func switchFromXtcpToTest() {
	fmt.Printf("%v : XTCP -> TEST\n", time.Now().UnixNano())
	flowMode = TEST_FROM_XTCP
	currMode = "TEST_FROM_XTCP"
	origFlowRate = flowRate
	switchTime = time.Now()
	return
}

func switchFromTestToXtcp() {
	fmt.Printf("%v : TEST -> XTCP\n", time.Now().UnixNano())
	flowMode = XTCP
	currMode = "XTCP"
	xtcpData.setXtcpCwnd(flowRate)
	switchTime = time.Now()
	return
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

		rttThresh := time.Duration(1.5*float64(min_rtt.Nanoseconds())) * time.Nanosecond

		if dZt > delayToTestThresh*est_bandwidth || rtt > rttThresh {
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

func updateRateTestFromDelay(rt float64) float64 {
	if elapsed := time.Since(switchTime); elapsed < min_rtt {
		// in the first min_rtt
		// increase rate by 50%
		return rt * 1.5
	} else if elapsed < 2*min_rtt {
		// in the second min_rtt
		// return to original rate
		return origFlowRate
	} else if elapsed < 3*min_rtt {
		// in the third min_rtt
		//decrease rate by ~50%
		return rt * 0.6
	}
	return origFlowRate
}

func updateRateTestFromXtcp(rt float64) float64 {
	if elapsed := time.Since(switchTime); elapsed < min_rtt {
		// in the first min_rtt
		//decrease rate by ~50%
		return rt * 0.6
	} else if elapsed < 2*min_rtt {
		// in the second min_rtt
		// return to original rate
		return origFlowRate
	} else if elapsed < 3*min_rtt {
		// in the third min_rtt
		// increase rate by 50%
		return rt * 1.5
	}
	return origFlowRate
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

func measure() (
	rin float64,
	rout float64,
	zt float64,
	avgRtt time.Duration,
	err error,
) {
	lv, err := rtts.Latest()
	if err != nil {
		return
	}
	rtt := time.Duration(lv.(durationLogVal))

	rout, oldPkt, newPkt, err := ThroughputFromTimes(
		ackTimes,
		time.Now(),
		rtt,
	)
	if err != nil {
		return
	}

	t1, t2, err := PacketTimes(sendTimes, oldPkt, newPkt)
	if err != nil {
		return
	}

	rin, _, _, err = ThroughputFromTimes(sendTimes, t1, t1.Sub(t2))
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
		fmt.Println(err)
		return
	}

	shouldSwitch(zt, rtt)
	zt_history.Add(time.Now(), floatLogVal(zt))

	flowRateLock.Lock()

	switch flowMode {
	case DELAY:
		flowRate = updateRateDelay(
			flowRate,
			est_bandwidth,
			rin,
			zt,
			rtt,
		)
	case XTCP:
		flowRate = xtcpData.updateRateXtcp(rtt)
	case TEST_FROM_DELAY:
		flowRate = updateRateTestFromDelay(flowRate)
	case TEST_FROM_XTCP:
		flowRate = updateRateTestFromXtcp(flowRate)
	}

	if flowRate < 0 {
		panic("negative flow rate")
	}

	flowRateLock.Unlock()

}
