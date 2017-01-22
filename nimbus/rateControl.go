package main

import (
	"fmt"
	"math"
	"time"
)

const (
	alpha = 1

	// switching parameters
	xtcpTimeout = 20 // rtts
)

type PulseMode int

const (
	UP_PULSE PulseMode = iota
	DOWN_PULSE
	PULSE_WAIT
)

var est_bandwidth float64

var beta float64

// regularly spaced measurements
var zt_history *TimedLog
var xt_history *TimedLog

// test state
var pulseMode PulseMode
var switchTime time.Time
var elasticityReset time.Time
var totElasticity float64

var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	est_bandwidth = 10e6

	zt_history = InitTimedLog(min_rtt)
	xt_history = InitTimedLog(min_rtt)

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33

	// TODO tracking
	maxQd = 2 * min_rtt

	switchTime = time.Now()
	elasticityReset = switchTime
	pulseMode = PULSE_WAIT
}

func deltaZt(zt float64, delay time.Duration) (float64, error) {
	zt_history.mux.Lock()
	oldZtVal, _, err := zt_history.Before(time.Now().Add(-1 * delay))
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

func deltaXt(from time.Time, rtt time.Duration, delay time.Duration) (float64, error) {
	xt_history.mux.Lock()
	oldXt, _, err := xt_history.Before(from.Add(-1 * delay))
	xt_history.mux.Unlock()
	if err != nil {
		return 0, err
	}

	return rtt.Seconds() - oldXt.(time.Duration).Seconds(), nil
}

func switchToDelay(rtt time.Duration) {
	fmt.Printf("%v : %s -> DELAY\n", time.Since(startTime), currMode)

	flowMode = DELAY
	currMode = "DELAY"
	switchTime = time.Now()
	return
}

func switchToXtcp(rtt time.Duration) {
	fmt.Printf("%v : %s -> XTCP\n", time.Since(startTime), currMode)

	flowMode = XTCP
	currMode = "XTCP"
	xtcpData.setXtcpCwnd(flowRate, rtt)
	switchTime = time.Now()
	return
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
	return
}

func integrateElasticity(zt float64, rtt time.Duration) {
	measurementInterval := time.Duration(10) * time.Millisecond

	xt_history.mux.Lock()
	oldXtVal, t, err := xt_history.Before(time.Now().Add(-1 * rtt))
	xt_history.mux.Unlock()
	if err != nil {
		err = fmt.Errorf("oldXt: %v", err)
		return
	}
	oldXt := oldXtVal.(time.Duration)

	dZt, err := deltaZt(zt, measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaZt: %v", err)
		return
	}

	dXt, err := deltaXt(t, oldXt, measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaXt: %v", err)
		return
	}

	elasticity := (dZt / est_bandwidth) * (dXt / min_rtt.Seconds())
	totElasticity += elasticity
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
		rin, rout, zt, rtt, err := measure(min_rtt)
		if err != nil {
			continue
		}

		yt := time.Duration(float64((rtt-min_rtt).Nanoseconds())*(rout/float64(est_bandwidth))) * time.Nanosecond

		avgYt, err := xt_history.AvgBetween(
			time.Now().Add(-1*min_rtt),
			time.Now(),
			yt,
			func(a TimedLogVal, b TimedLogVal) TimedLogVal {
				// sum
				return a.(time.Duration) + b.(time.Duration)
			},
			func(a TimedLogVal, n int) TimedLogVal {
				// div
				return time.Duration(a.(time.Duration).Nanoseconds() / int64(n))
			},
		)

		if err != nil {
			xt_history.Add(time.Now(), yt)
		} else {
			xt_history.Add(time.Now(), avgYt)
		}

		zt_history.Add(time.Now(), zt)

		integrateElasticity(zt, rtt)

		fmt.Printf("%v : %v %v %v %v %v %v\n", time.Since(startTime), zt, rtt, rin, rout, totElasticity, flowRate)
		<-time.After(tick)
	}
}

func changePulses(fr float64, rtt time.Duration) float64 {
	if time.Since(switchTime) < min_rtt {
		return fr
	}

	switch pulseMode {
	case UP_PULSE:
		pulseMode = DOWN_PULSE
		return fr * 1.25
	case DOWN_PULSE:
		pulseMode = UP_PULSE
		return fr * 0.75
	}

	switchTime = time.Now()
	return fr
}

func doUpdate() {
	rin, _, zt, rtt, err := measure(min_rtt)
	if err != nil {
		return
	}

	fmt.Println("ELAST", time.Since(elasticityReset), totElasticity)
	if time.Since(elasticityReset) > 3*min_rtt && totElasticity < -0.02 {
		if flowMode == DELAY {
			switchToXtcp(rtt)
		}
		totElasticity = 0
		elasticityReset = time.Now()
	} else if time.Since(elasticityReset) > time.Duration(3)*time.Second {
		if flowMode == XTCP {
			switchToDelay(rtt)
		}
		elasticityReset = time.Now()
		totElasticity = 0
	}

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
	}

	flowRate = changePulses(flowRate, rtt)

	if flowRate < 0 {
		panic("negative flow rate")
	}

	flowRateLock.Unlock()

}
