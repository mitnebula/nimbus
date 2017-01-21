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
	xtcpTimeout = 20 // rtts
)

var beta float64

// regularly spaced measurements
var zt_history *TimedLog
var xt_history *TimedLog

// test state
var delayToTestThresh float64
var switchTime time.Time
var testTimeout time.Duration
var origFlowRate float64
var testPulses int
var testResultXtcp bool

var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	zt_history = InitTimedLog(min_rtt)
	xt_history = InitTimedLog(min_rtt)
	switchTime = time.Now()

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33
	origFlowRate = flowRate

	// TODO tracking
	maxQd = 2 * min_rtt
	testTimeout = maxQd
	delayToTestThresh = 0.05 * est_bandwidth

	testResultXtcp = false
}

func deltaZt(zt float64, rtt time.Duration) (float64, error) {
	zt_history.mux.Lock()
	oldZtVal, _, err := zt_history.Before(time.Now().Add(-1 * rtt))
	zt_history.mux.Unlock()
	if err != nil {
		return 0, err
	}

	oldZt := oldZtVal.(float64)

	if zt == 0 || oldZt == 0 {
		// zt is invalid
		return 0, fmt.Errorf("invalid zt")
	}

	return zt - oldZt, nil
}

func deltaXt(rtt time.Duration) (float64, error) {
	oldXt, _, err := xt_history.Before(time.Now().Add(-1 * rtt))
	if err != nil {
		return 0, err
	}

	return rtt.Seconds() - oldXt.(time.Duration).Seconds(), nil
}

func switchFromTestToDelay(rtt time.Duration) {
	fmt.Printf("%v : %s -> DELAY\n", time.Since(startTime), currMode)

	flowMode = DELAY
	currMode = "DELAY"
	flowRate = origFlowRate
	switchTime = time.Now()
	return
}

func switchToTest(zt float64, rtt time.Duration) {
	testResultXtcp = false
	if rtt.Seconds() < min_rtt.Seconds()*1.25 {
		return
	} else if rtt.Seconds() > 0.25*min_rtt.Seconds()+maxQd.Seconds() {
		return
	}

	rttThresh := min_rtt + maxQd/2
	testPulses = 10
	if rtt > rttThresh {
		flowMode = TEST_HIGH_RTT_DOWN_PULSE
		delayToTestThresh = zt * (0.5 * min_rtt.Seconds()) / (rtt.Seconds())
		delayToTestThresh = math.Max(delayToTestThresh, 0.05*est_bandwidth) / 2
		currMode = "TEST_HIGH_RTT"
	} else {
		flowMode = TEST_LOW_RTT_UP_PULSE
		delayToTestThresh = zt * (0.5 * min_rtt.Seconds()) / (rtt.Seconds() + 0.5*min_rtt.Seconds())
		delayToTestThresh = math.Max(delayToTestThresh, 0.05*est_bandwidth) / 2
		currMode = "TEST_LOW_RTT"
	}
	origFlowRate = flowRate
	min_rate := ONE_PACKET / min_rtt.Seconds()
	to := (1 + math.Max(1, (est_bandwidth/2)/(origFlowRate-min_rate))) * float64(testPulses)
	testTimeout = time.Duration(int64(to*float64(min_rtt.Nanoseconds()))) + 3*rtt
	// TODO measure rin and rout over min_rtt, change above to 3min_rtt + 2 rtt

	fmt.Printf("%v : %s -> TEST %v\n", time.Since(startTime), currMode, delayToTestThresh)
	switchTime = time.Now()
	return
}

func switchFromTestToXtcp(rtt time.Duration) {
	fmt.Printf("%v : %s -> XTCP\n", time.Since(startTime), currMode)

	flowMode = XTCP
	currMode = "XTCP"
	flowRate = origFlowRate
	xtcpData.setXtcpCwnd(flowRate, rtt)
	switchTime = time.Now()
	return
}

func shouldSwitch(zt float64, rtt time.Duration) {
	elapsed := time.Since(switchTime)
	if elapsed < 3*min_rtt || zt == 0 {
		return
	}

	switch flowMode {
	case DELAY:
		if elapsed > xtcpTimeout*min_rtt {
			switchToTest(zt, rtt)
		}

		// if delta zt > alpha * mu
		// go to test
		dZt, err := deltaZt(zt, rtt)
		if err != nil {
			return
		}

		if dZt > delayToTestThresh*est_bandwidth {
			switchToTest(zt, rtt)
			return
		}

		// else if rtt > thresh and is increasing
		// go to test
		rttThresh := time.Duration(1.5*float64(min_rtt.Nanoseconds())) * time.Nanosecond
		dXt, err := deltaXt(rtt)
		if err != nil {
			return
		}

		if rtt > rttThresh && dXt > 0 {
			switchToTest(zt, rtt)
			return
		}
		break
	case XTCP:
		// if timeout expires
		// go to test
		if elapsed > xtcpTimeout*min_rtt {
			switchToTest(zt, rtt)
		}
		break
	case TEST_WAIT:
		// if timeout expires
		// go to delay

		if elapsed > testTimeout {
			if testResultXtcp {
				switchFromTestToXtcp(rtt)
			} else {
				switchFromTestToDelay(rtt)
			}
		} else if testResultXtcp {
			return
		}

		// if delta zt > alpha * mu
		// go to xtcp
		dZt, err := deltaZt(zt, rtt)
		if err != nil {
			return
		}

		if dZt < -1*delayToTestThresh {
			//fmt.Println(time.Since(startTime), dZt, delayToTestThresh, rtt, zt)
			testResultXtcp = true
		}
		break
	}
}

func updateRateTestUpPulse(rt float64) float64 {
	return origFlowRate + est_bandwidth/2
}

func updateRateTestDownPulse(rt float64) float64 {
	min_rate := ONE_PACKET / min_rtt.Seconds()
	return math.Max(origFlowRate-est_bandwidth/2, min_rate)
}

func updateRateTestWait(rt float64) float64 {
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

	minRate := float64(ONE_PACKET) / min_rtt.Seconds() // send at least 1 packet per rtt
	if newRate < minRate || math.IsNaN(newRate) {
		newRate = minRate
	}

	return newRate
}

func measure(interval time.Duration) (
	rin float64,
	rout float64,
	zt float64,
	rtt time.Duration,
	err error,
) {
	lv, err := rtts.Latest()
	if err != nil {
		return
	}
	rtt = time.Duration(lv.(durationLogVal))
	rout, oldPkt, newPkt, err := ThroughputFromTimes(
		ackTimes,
		time.Now(),
		interval,
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
	if rtt.Seconds() < 1.05*min_rtt.Seconds() {
		zt = 0
	}
	if zt < 0 {
		zt = 0
	}

	//fmt.Printf("%v : %v %v %v %v\n", time.Since(startTime), zt, rtt, rin, rout)
	return
}

func flowRateUpdater() {
	for {
		untilNextUpdate = time.Duration(0)
		doUpdate()
		if untilNextUpdate == time.Duration(0) {
			untilNextUpdate = time.Duration(10) * time.Millisecond
		}

		if time.Now().After(endTime) {
			doExit()
		}
		<-time.After(untilNextUpdate)
	}
}

func measurePeriod() {
	tick := time.Duration(10) * time.Millisecond
	for {
		rin, rout, zt, rtt, err := measure(2 * min_rtt)
		if err != nil {
			continue
		}

		zt_history.Add(time.Now(), zt)
		xt_history.Add(time.Now(), rtt)

		xt_history.mux.Lock()
		old_xt, _, err := xt_history.Before(time.Now().Add(-1 * rtt))
		xt_history.mux.Unlock()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("%v : %v %v %v %v %v\n", time.Since(startTime), zt, rtt, rin, rout, old_xt)
		<-time.After(tick)
	}
}

func doUpdate() {
	lv, err := rtts.Latest()
	if err != nil {
		fmt.Println(err)
		return
	}
	rtt := time.Duration(lv.(durationLogVal))

	rin, _, zt, _, err := measure(rtt)
	if err != nil {
		return
	}
	flowRateLock.Lock()

	shouldSwitch(zt, rtt)

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

	case TEST_LOW_RTT_UP_PULSE:
		flowRate = updateRateTestUpPulse(flowRate)
		untilNextUpdate = min_rtt
		flowMode = TEST_LOW_RTT_DOWN_PULSE

	case TEST_LOW_RTT_DOWN_PULSE:
		flowRate = updateRateTestDownPulse(flowRate)
		min_rate := ONE_PACKET / min_rtt.Seconds()
		to := math.Max(1, (est_bandwidth/2)/(origFlowRate-min_rate))
		untilNextUpdate = time.Duration(int64(to * float64(min_rtt.Nanoseconds())))
		if testPulses <= 0 {
			flowMode = TEST_WAIT
		} else {
			testPulses--
			flowMode = TEST_LOW_RTT_UP_PULSE
		}

	case TEST_HIGH_RTT_UP_PULSE:
		flowRate = updateRateTestUpPulse(flowRate)
		untilNextUpdate = min_rtt
		if testPulses <= 0 {
			flowMode = TEST_WAIT
		} else {
			testPulses--
			flowMode = TEST_HIGH_RTT_DOWN_PULSE
		}

	case TEST_HIGH_RTT_DOWN_PULSE:
		flowRate = updateRateTestDownPulse(flowRate)
		min_rate := ONE_PACKET / min_rtt.Seconds()
		to := math.Max(1, (est_bandwidth/2)/(origFlowRate-min_rate))
		untilNextUpdate = time.Duration(int64(to * float64(min_rtt.Nanoseconds())))
		flowMode = TEST_HIGH_RTT_UP_PULSE

	case TEST_WAIT:
		flowRate = updateRateTestWait(flowRate)
	}

	if flowRate < 0 {
		panic("negative flow rate")
	}

	flowRateLock.Unlock()

}
