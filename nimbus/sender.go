package main

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

const (
	est_bandwidth = 10e6
	alpha         = 1

	// rate threshold before becoming more aggressive
	rate_thresh = 0.9 // units: factor of rin from 500 updates ago. TODO set properly
)

var flowRate float64
var flowRateLock sync.Mutex

var min_rtt time.Duration
var beta float64

// Log is thread-safe
var rtts *Log
var sendTimes *TimedLog
var ackTimes *TimedLog
var rin_history *Log

type Mode int

const (
	DELAY Mode = iota
	BETAZERO
	XTCP
)

var flowMode Mode

// beta zero mode timeout
var betaZeroTimeout int64

func init() {
	flowMode = DELAY

	flowRate = 0.82 * 1e7
	min_rtt = time.Duration(999) * time.Hour

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (est_bandwidth / 0.001) * 0.4

	rtts = InitLog(900)
	sendTimes = InitTimedLog(min_rtt, 0)
	ackTimes = InitTimedLog(min_rtt, 0)
	rin_history = InitLog(500)
}

func Sender(ip string, port string) error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return err
	}

	err = conn.SetWriteBuffer(SOCK_BUF_SIZE)
	if err != nil {
		fmt.Println("err setting sock wr buf sz", err)
	}

	recvExit := make(chan interface{})
	rtt_history := make(chan int64)

	go handleAck(conn, addr, rtt_history, recvExit)
	go rttUpdater(rtt_history)
	go flowRateUpdater()
	go output()

	go send(conn, recvExit)

	<-recvExit

	return nil
}

func output() {
	for _ = range time.Tick(2 * time.Second) {
		tpt, err := ThroughputFromTimes(ackTimes, 0)
		if err != nil {
			continue
		}
		fmt.Printf("%v : %v\n", Now(), tpt)
	}
}

// keep rtt up to date (from received acks)
func rttUpdater(rtt_history chan int64) {
	for t := range rtt_history {
		rtt := time.Duration(t) * time.Nanosecond
		if rtt < min_rtt {
			min_rtt = rtt
			beta = (est_bandwidth / min_rtt.Seconds()) * 0.4
		}

		sendTimes.UpdateDuration(rtt)
		ackTimes.UpdateDuration(rtt)
		sendTimes.UpdateSlack(rtt * 2)
		ackTimes.UpdateSlack(rtt * 2)

		rtts.Add(durationLogVal(rtt))
	}
}

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
	} else if flowMode == XTCP && rin < est_bandwidth-zt {
		fmt.Println(Now(), "XTCP -> BETAZERO")
		flowMode = BETAZERO
		betaZeroTimeout = Now() + rtt.Nanoseconds()*4
	}
}

func updateRateDelay(
	flowRate float64,
	est_bandwidth float64,
	rin float64,
	zt float64,
	rtt time.Duration,
) float64 {
	var newRate float64
	switch flowMode {
	case DELAY:
		newRate = rin + alpha*(est_bandwidth-zt-rin) - beta*(rtt.Seconds()-(1.1*min_rtt.Seconds()))
	case BETAZERO:
		newRate = rin + alpha*(est_bandwidth-zt-rin)
	}

	minRate := 1490 * 8.0 / min_rtt.Seconds() // send at least 1 packet per rtt
	if newRate < minRate || math.IsNaN(newRate) {
		newRate = minRate
	}
	//fmt.Printf("time: %v old_rate: %f curr_rate: %f rin: %f zt: %f min_rtt: %v curr_rtt: %v\n", Now(), flowRate, newRate, rin, zt, min_rtt, rtt)
	return newRate
}

func flowRateUpdater() {
	for {
		var wait time.Duration
		// update rate every ~rtt
		if rtts.Len() > 0 {
			lv, _ := rtts.Latest()
			wait = time.Duration(lv.(durationLogVal)) / 5
		} else {
			wait = time.Duration(5) * time.Millisecond
		}
		<-time.After(wait)

		lv, err := rtts.Latest()
		if err != nil {
			continue
		}
		rtt := time.Duration(lv.(durationLogVal))

		rin, err := ThroughputFromTimes(sendTimes, rtt)
		if err != nil {
			fmt.Println(err, sendTimes.Len(), CurrSpan(sendTimes.times), rtt)
			continue
		}

		rin_history.Add(floatLogVal(rin))

		rout, err := ThroughputFromTimes(ackTimes, 0)
		if err != nil {
			continue

		}

		zt := est_bandwidth*(rin/rout) - rin

		//shouldSwitch(zt, rtt)

		flowRateLock.Lock()

		switch flowMode {
		case BETAZERO:
			fallthrough
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

// read the current flow rate and set the pacing channel appropriately
func flowPacer(pacing chan interface{}) {
	for { // cannot use time.Tick because tick interval is dynamic
		//flowRateLock.Lock()
		waitSeconds := 1e9 * 1500 * 8.0 / flowRate // nanoseconds to wait until next packet
		wt := time.Duration(waitSeconds) * time.Nanosecond
		//flowRateLock.Unlock()
		<-time.After(wt)
		pacing <- struct{}{}
	}
}

func send(
	conn *net.UDPConn,
	done chan interface{},
) error {
	pacing := make(chan interface{})
	go flowPacer(pacing)

	for {
		seq, vfid := xtcpData.getNextSeq()

		pkt := Packet{
			SeqNo:   seq,
			VirtFid: vfid,
			Echo:    Now(),
			Payload: "",
		}

		//fmt.Println("sending ", Now(), pkt.VirtFid, pkt.SeqNo)

		sendTimes.Add(time.Now(), intLogVal(Now()))
		err := SendPacket(conn, pkt, 1480)
		if err != nil {
			fmt.Println(err)
		}

		<-pacing
	}

	//done <- struct{}{}
	//return nil
}

func handleAck(
	conn *net.UDPConn,
	expSrc *net.UDPAddr,
	rtt_history chan int64,
	done chan interface{},
) {
	for {
		pkt, srcAddr, err := RecvPacket(conn)
		if err != nil {
			fmt.Println(err)
		}
		if fmt.Sprintf("%s", srcAddr) != fmt.Sprintf("%s", expSrc) {
			fmt.Println(fmt.Errorf("got packet from unexpected src: %s; expected %s", srcAddr, expSrc))
		}

		//fmt.Println("recvdAck", Now(), pkt.VirtFid, pkt.SeqNo)

		ackTimes.Add(time.Now(), intLogVal(Now()))

		// check for XTCP packet drop
		if ok := xtcpData.checkXtcpSeq(pkt.VirtFid, pkt.SeqNo); !ok {
			xtcpData.dropDetected(pkt.VirtFid)
		} else {
			xtcpData.increaseXtcpWind(pkt.VirtFid)
		}

		delay := pkt.RecvTime - pkt.Echo
		rtt_history <- delay * 2 // multiply one-way delay by 2
	}
}
