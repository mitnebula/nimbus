package main

import (
	log "github.com/Sirupsen/logrus"
	"math"
	"time"

	"github.com/akshayknarayan/history"
)

const (
	alpha = 0.8
)

var est_bandwidth float64
var useSwitching bool
var delayThreshold float64

var beta float64

// regularly spaced measurements
var zt_history *history.History
var xt_history *history.History

var modeSwitchTime time.Time

// test state
var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	est_bandwidth = 10e6

	zt_history = history.MakeHistory(min_rtt)
	xt_history = history.MakeHistory(min_rtt)

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33

	// TODO tracking
	maxQd = min_rtt

	modeSwitchTime = time.Now()
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
	if zt < 0 {
		zt = 0
	}

	qd := float64((rtt - min_rtt).Nanoseconds())
	ytNs := qd * (rout / float64(est_bandwidth))
	yt := ytNs / qd

	if zt > 0 {
		zt_history.Add(time.Now(), zt)
	}
	xt_history.Add(time.Now(), yt)

	log.WithFields(log.Fields{
		"elapsed":  time.Since(startTime),
		"zt":       zt,
		"rtt":      rtt,
		"rin":      rin,
		"rout":     rout,
		"flowRate": flowRate,
		"yt":       yt,
	}).Debug()
	return
}

func changePulses(fr float64) float64 {
	elapsed := time.Since(startTime).Seconds()
	return fr + (*pulseSize)*fr*math.Sin((1/(2*min_rtt.Seconds()))*2*math.Pi*elapsed)
}

func shouldSwitch(rtt time.Duration) {
	// not implemented
	// TODO FFT-based switching logic
	return
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

	flowRate = changePulses(flowRate)

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
