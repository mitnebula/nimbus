package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"math"
	"time"

	"github.com/akshayknarayan/history"
)

const (
	alpha = 0.8

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
var useSwitching bool
var delayThreshold float64

var beta float64

// regularly spaced measurements
var zt_history *history.History
var xt_history *history.History
var esty_history *history.History

var modeSwitchTime time.Time

// test state
var pulseMode PulseMode
var pulseSwitchTime time.Time
var numPulses = 0

var totElasticity float64

var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	est_bandwidth = 10e6

	zt_history = history.MakeHistory(min_rtt)
	xt_history = history.MakeHistory(min_rtt)
	esty_history = history.MakeHistory(time.Duration(15) * time.Second)

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33

	// TODO tracking
	maxQd = min_rtt

	pulseSwitchTime = time.Now()
	modeSwitchTime = time.Now()
	pulseMode = PULSE_WAIT
}

func deltaZt(zt float64, delay time.Duration) (float64, error) {
	oldZtVal, _, err := zt_history.Before(time.Now().Add(-1 * delay))
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
	oldXt, _, err := xt_history.Before(from.Add(-1 * delay))
	if err != nil {
		return 0, err
	}

	return rtt.Seconds() - oldXt.(time.Duration).Seconds(), nil
}

func deltaYt(from time.Time, yt float64, delay time.Duration) (float64, error) {
	oldYt, _, err := xt_history.Before(from.Add(-1 * delay))
	if err != nil {
		return 0, err
	}

	return yt - oldYt.(float64), nil
}

func switchToDelay(rtt time.Duration) {
	if flowMode == DELAY {
		return
	}

	log.WithFields(log.Fields{
		"elapsed": time.Since(startTime),
		"from":    currMode,
		"to":      "DELAY",
	}).Info("switched mode")

	flowMode = DELAY
	currMode = "DELAY"
	pulseMode = PULSE_WAIT
	modeSwitchTime = time.Now()
}

func switchToXtcp(rtt time.Duration) {
	if flowMode == XTCP {
		return
	}

	log.WithFields(log.Fields{
		"elapsed": time.Since(startTime),
		"from":    currMode,
		"to":      "XTCP",
	}).Info("switched mode")

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
	beta = 0.5
	delta := rtt.Seconds()
	newRate := rin + alpha*(est_bandwidth-zt-rin) - ((est_bandwidth*beta)/delta)*(rtt.Seconds()-(delayThreshold*min_rtt.Seconds()))

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
	rtt = lv.(time.Duration)
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

	qd := float64((rtt - min_rtt).Nanoseconds())
	ytNs := qd * (rout / float64(est_bandwidth))
	yt := ytNs / qd

	if zt > 0 {
		zt_history.Add(time.Now(), zt)
	}
	xt_history.Add(time.Now(), yt)

	elast := integrateElasticity(zt, rtt)
	oldElast, _, err := esty_history.Before(time.Now().Add(-1 * (time.Duration(5) * time.Second)))
	if err == nil {
		elast -= oldElast.(float64)
	}

	oldXtVal, _, err := xt_history.Before(time.Now().Add(-1 * interval))
	if err != nil {
		return
	}
	oldXt := oldXtVal.(float64)

	log.WithFields(log.Fields{
		"elapsed":    time.Since(startTime),
		"zt":         zt,
		"rtt":        rtt,
		"rin":        rin,
		"rout":       rout,
		"elast_5sec": elast,
		"flowRate":   flowRate,
		"yt":         yt,
		"oldYt":      oldXt,
	}).Debug()
	return
}

func integrateElasticity(zt float64, rtt time.Duration) float64 {
	var totEsty float64

	dZt, err := deltaZt(zt, *measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaZt: %v", err)
		return 0
	}

	oldYtVal, t, err := xt_history.Before(time.Now().Add(-1 * rtt))
	if err != nil {
		err = fmt.Errorf("oldYt: %v", err)
		return 0
	}
	oldYt := oldYtVal.(float64)

	dYt, err := deltaYt(t, oldYt, *measurementInterval)
	if err != nil {
		err = fmt.Errorf("deltaYt: %v", err)
		return 0
	}

	elasticity := (dZt / est_bandwidth) * dYt

	lv, _, err := esty_history.Before(time.Now())
	if err != nil {
		esty_history.Add(time.Now(), elasticity)
		return elasticity
	} else {
		totEsty = lv.(float64) + elasticity
	}

	esty_history.Add(time.Now(), totEsty)
	return totEsty
}

func changePulses(fr float64, rtt time.Duration) float64 {
	if time.Since(pulseSwitchTime) < min_rtt {
		return fr
	}

	switch pulseMode {
	case PULSE_WAIT:
		numPulses = 2
		if rtt > min_rtt+maxQd/2 {
			pulseMode = UP_PULSE
			pulseSwitchTime = time.Now()
		} else {
			pulseMode = DOWN_PULSE
			pulseSwitchTime = time.Now()
		}
		return fr
	case UP_PULSE:
		if numPulses == 0 {
			pulseMode = PULSE_WAIT
			pulseSwitchTime = time.Now()
			return fr / (1 + *pulseSize)
		} else {
			numPulses--
			pulseMode = DOWN_PULSE
			pulseSwitchTime = time.Now()
			return fr * (1 - *pulseSize) / (1 + *pulseSize)
		}
	case DOWN_PULSE:
		if numPulses == 0 {
			pulseMode = PULSE_WAIT
			pulseSwitchTime = time.Now()
			return fr / (1 - *pulseSize)
		} else {
			numPulses--
			pulseMode = UP_PULSE
			pulseSwitchTime = time.Now()
			return fr * (1 + *pulseSize) / (1 - *pulseSize)
		}
	default:
		err := fmt.Errorf("unknown pulse mode: %v", pulseMode)
		log.Panic(err)
		return -1
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
		log.WithFields(log.Fields{
			"elapsed": time.Since(startTime),
			"rtt":     rtt,
			"thresh":  delayThreshold * min_rtt.Seconds(),
		}).Debug("rtt too low")
		return
	} else if r > min_rtt.Seconds()+0.5*maxQd.Seconds() {
		log.WithFields(log.Fields{
			"elapsed": time.Since(startTime),
			"rtt":     rtt,
			"thresh":  min_rtt.Seconds() + 0.5*maxQd.Seconds(),
		}).Debug("rtt too big")
		switchToXtcp(rtt)
		return
	}

	totElastVal, _, err := esty_history.Before(time.Now())
	if err != nil {
		log.Error(err)
		return
	}

	totElast := totElastVal.(float64)

	sec5 := elasticityWindow(totElast, time.Duration(5)*time.Second)
	sec2 := elasticityWindow(totElast, time.Duration(2)*time.Second)
	minrtt10 := elasticityWindow(totElast, 10*min_rtt)

	log.WithFields(log.Fields{
		"elapsed":        time.Since(startTime),
		"elast_5sec":     sec5,
		"elast_2sec":     sec2,
		"elast_10minrtt": minrtt10,
	}).Debug("ELASTICITY")

	if sec2 > 0 {
		log.WithFields(log.Fields{
			"elapsed":    time.Since(startTime),
			"elast_2sec": sec2,
		}).Debug("elast above thresh")
		switchToDelay(rtt)
		return
	} else if sec5 < -0.1 {
		log.WithFields(log.Fields{
			"elapsed":    time.Since(startTime),
			"elast_5sec": sec5,
		}).Debug("elast below thresh")
		switchToXtcp(rtt)
		return
	}
}

func doUpdate() {
	rin, _, zt, rtt, err := measure(time.Duration(*measurementTimescale*min_rtt.Nanoseconds()) * time.Nanosecond)
	if err != nil {
		return
	}

	shouldSwitch(rtt)

	flowRateLock.Lock()
	defer flowRateLock.Unlock()

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
		log.Panic("negative flow rate")
	}
}

func flowRateUpdater() {
	for _ = range time.Tick(*measurementInterval) {
		doUpdate()

		if time.Now().After(endTime) {
			doExit()
		}
	}
}
