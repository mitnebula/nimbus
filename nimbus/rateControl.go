package main

import (
	"fmt"
	"math"
	"time"
)

func shouldSwitch(zt float64, rtt time.Duration) {
	oldest, newest, err := rin_history.Ends()
	if err != nil {
		fmt.Println(err)
		return
	}

	old_rin := float64(oldest.(floatLogVal))
	rin := float64(newest.(floatLogVal))

	if flowMode != XTCP && flowRate < old_rin*0.9 {
		if flowMode == DELAY {
			fmt.Println(Now(), "DELAY -> BETAZERO")
			flowMode = BETAZERO
		} else if flowMode == BETAZERO {
			fmt.Println(Now(), "BETAZERO -> XTCP")
			xtcpData.switchToXtcp(flowRate)
		}
	} else if flowMode == BETAZERO && Now() > betaZeroTimeout {
		fmt.Println(Now(), "BETAZERO -> DELAY")
		flowMode = DELAY
	} else if flowMode == XTCP && rin < flowRate-zt {
		fmt.Println(Now(), "XTCP -> BETAZERO")
		flowMode = BETAZERO
		betaZeroTimeout = Now() + rtt.Nanoseconds()*4
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
	newRate := rin + alpha*(est_bandwidth-zt-rin) - beta*(rtt.Seconds()-(1.25*min_rtt.Seconds()))

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

		//fmt.Printf("time: %v rtt: %v/%v rin: %.3v rout: %.3v zt: %.3v\n", Now(), rtt, min_rtt, rin, rout, zt)

		//shouldSwitch(zt, rtt)

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
}
