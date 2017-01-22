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
var esty_history *TimedLog

var modeSwitchTime time.Time

// test state
var pulseMode PulseMode
var pulseSwitchTime time.Time
var totElasticity float64
var numPulses int

var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	est_bandwidth = 10e6

	zt_history = InitTimedLog(min_rtt)
	xt_history = InitTimedLog(min_rtt)
	esty_history = InitTimedLog(time.Duration(15) * time.Second)

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33

	// TODO tracking
	maxQd = 2 * min_rtt

	pulseSwitchTime = time.Now()
	modeSwitchTime = time.Now()
	pulseMode = PULSE_WAIT
	numPulses = 2
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
	if flowMode == DELAY {
		return
	}

	fmt.Printf("%v : %s -> DELAY\n", time.Since(startTime), currMode)

	flowMode = DELAY
	currMode = "DELAY"
	pulseMode = PULSE_WAIT
	modeSwitchTime = time.Now()
}

func switchToXtcp(rtt time.Duration) {
	if flowMode == XTCP {
		return
	}

	fmt.Printf("%v : %s -> XTCP\n", time.Since(startTime), currMode)

	flowMode = XTCP
	currMode = "XTCP"
	pulseMode = PULSE_WAIT
	xtcpData.setXtcpCwnd(flowRate, rtt)
	modeSwitchTime = time.Now()
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

	if rout > est_bandwidth {
		rout = est_bandwidth
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
	if zt > est_bandwidth {
		zt = est_bandwidth
	}
	return
}

func integrateElasticity(zt float64, rtt time.Duration) float64 {
	var totEsty float64
	measurementInterval := time.Duration(10) * time.Millisecond

	xt_history.mux.Lock()
	oldXtVal, t, err := xt_history.Before(time.Now().Add(-1 * rtt))
	xt_history.mux.Unlock()
	if err != nil {
		err = fmt.Errorf("oldXt: %v", err)
		return 0
	}
	oldXt := oldXtVal.(time.Duration)

	dZt, err := deltaZt(zt, measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaZt: %v", err)
		return 0
	}

	dXt, err := deltaXt(t, oldXt, measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaXt: %v", err)
		return 0
	}

	elasticity := (dZt / est_bandwidth) * (dXt / min_rtt.Seconds())

	esty_history.mux.Lock()
	lv, _, err := esty_history.Before(time.Now())
	esty_history.mux.Unlock()
	if err != nil {
		esty_history.Add(time.Now(), elasticity)
		return elasticity
	} else {
		totEsty = lv.(float64) + elasticity
	}

	esty_history.Add(time.Now(), totEsty)
	return totEsty
	//totElasticity += elasticity
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

		if zt > 0 {
			zt_history.Add(time.Now(), zt)
		}
		xt_history.Add(time.Now(), yt)

		elast := integrateElasticity(zt, rtt)
		esty_history.mux.Lock()
		oldElast, _, err := esty_history.Before(time.Now().Add(-1 * (time.Duration(5) * time.Second)))
		if err == nil {
			elast -= oldElast.(float64)
		}

		fmt.Printf("%v : %v %v %v %v %v %v %v\n", time.Since(startTime), zt, rtt, rin, rout, elast, flowRate, yt)
		esty_history.mux.Unlock()
		<-time.After(tick)
	}
}

func changePulses(fr float64, rtt time.Duration) float64 {
	if time.Since(pulseSwitchTime) < min_rtt {
		return fr
	}

	switch pulseMode {
	case PULSE_WAIT:
		if time.Since(pulseSwitchTime) < 1*min_rtt {
			return fr
		}

		numPulses = 10
		if rtt > min_rtt+maxQd/2 {
			pulseMode = UP_PULSE
			pulseSwitchTime = time.Now()
		} else {
			pulseMode = DOWN_PULSE
			pulseSwitchTime = time.Now()
		}
		return fr
	case UP_PULSE:
		if numPulses <= 1 {
			pulseMode = PULSE_WAIT
		} else {
			numPulses--
			pulseMode = DOWN_PULSE
		}
		pulseSwitchTime = time.Now()
		return fr * 1.5
	case DOWN_PULSE:
		if numPulses <= 1 {
			pulseMode = PULSE_WAIT
		} else {
			numPulses--
			pulseMode = UP_PULSE
		}
		pulseSwitchTime = time.Now()
		return fr * 0.5
	default:
		err := fmt.Errorf("unknown pulse mode: %v", pulseMode)
		panic(err)
	}
}

func elasticityWindow(tot float64, wind time.Duration) float64 {
	oldElast := float64(0)
	oldElastVal, _, err := esty_history.Before(time.Now().Add(-1 * wind))
	if err != nil {
		oldElast = 0
	} else {
		oldElast = oldElastVal.(float64)
	}

	return tot - oldElast
}

func shouldSwitch(rtt time.Duration) {
	modeSwitchTimeHorizon := modeSwitchTime.Add(3 * min_rtt)
	// if within 3 min_rtt of mode switch, do nothing
	if time.Now().Before(modeSwitchTimeHorizon) {
		return
	}

	// can't test properly if rtt too high or low
	if r := rtt.Seconds(); r < 1.25*min_rtt.Seconds() {
		fmt.Println("rtt too low", rtt, 1.25*min_rtt.Seconds())
		return
	} else if r > min_rtt.Seconds()+0.5*maxQd.Seconds() {
		fmt.Println("rtt too big", rtt, min_rtt.Seconds()+0.5*maxQd.Seconds())
		switchToXtcp(rtt)
		return
	}

	esty_history.mux.Lock()
	defer esty_history.mux.Unlock()
	totElastVal, _, err := esty_history.Before(time.Now())
	if err != nil {
		fmt.Println(err)
		return
	}

	totElast := totElastVal.(float64)

	sec5 := elasticityWindow(totElast, time.Duration(5)*time.Second)
	sec2 := elasticityWindow(totElast, time.Duration(2)*time.Second)
	minrtt10 := elasticityWindow(totElast, 10*min_rtt)

	fmt.Printf("ELASTICITY: %v %v %v %v\n", time.Since(startTime), sec5, sec2, minrtt10)

	if sec2 > 0 {
		fmt.Println("delay, elast not low long term", sec2)
		switchToDelay(rtt)
		return
	} else if sec5 < -0.1 {
		fmt.Println("xtcp, elast low long term", sec5)
		switchToXtcp(rtt)
		return
	}
}

func doUpdate() {
	rin, _, zt, rtt, err := measure(min_rtt)
	if err != nil {
		return
	}

	shouldSwitch(rtt)

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
