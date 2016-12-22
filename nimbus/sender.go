package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const est_bandwidth = 10e6
const alpha = 1

//const beta = 5.5 * est_bandwidth
const beta = 2.0 * est_bandwidth

var flowRate float64
var flowRateLock sync.Mutex

var min_rtt time.Duration

var throughput float64
var throughputLock sync.Mutex

// Log is thread-safe
var rtts *Log
var sendTimes *Log
var ackTimes *Log

func init() {
	flowRate = 0.82 * 1e7
	min_rtt = time.Duration(999) * time.Hour

	rtts = InitLog(900)
	sendTimes = InitLog(500)
	ackTimes = InitLog(500)
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

	err = conn.SetWriteBuffer(2000 * 1500)
	if err != nil {
		fmt.Println("err setting sock wr buf sz", err)
	}

	recvExit := make(chan interface{})
	rtt_history := make(chan int64)
	throughput_history := make(chan float64)

	go handleAck(conn, addr, rtt_history, throughput_history, recvExit)
	go rttUpdater(rtt_history)
	go throughputUpdater(throughput_history)
	go flowRateUpdater()

	go send(conn)

	<-recvExit

	return nil
}

// keep rtt up to date (from received acks)
func rttUpdater(rtt_history chan int64) {
	for t := range rtt_history {
		rtt := time.Duration(t) * time.Nanosecond
		if rtt < min_rtt {
			min_rtt = rtt
		}
		rtts.Add(LogDuration(rtt))
		//fmt.Println("rtt", rtt.Seconds())
	}
}

// keep r_out up to date (from received acks)
func throughputUpdater(throughput_history chan float64) {
	for t := range throughput_history {
		throughputLock.Lock()
		throughput = t
		throughputLock.Unlock()
		//fmt.Println("tpt", throughput)
	}
}

func updateRateDelay(
	flowRate float64,
	est_bandwidth float64,
	rin float64,
	zt float64,
	rtt time.Duration,
) float64 {
	//min_rtt := MinRtt(rtts)
	newRate := rin + alpha*(est_bandwidth-zt-rin) - beta*(rtt.Seconds()-(1.1*min_rtt.Seconds()))
	minRate := 1490 * 8.0 / min_rtt.Seconds() // send at least 1 packet per rtt
	if newRate < minRate {
		newRate = minRate
	}
	fmt.Printf("time: %v old_rate: %f curr_rate: %f rin: %f zt: %f min_rtt: %v curr_rtt: %v\n", Now(), flowRate, newRate, rin, zt, min_rtt, rtt)
	return newRate
}

func flowRateUpdater() {
	for {
		var wait time.Duration
		// update rate every ~rtt
		if rtts.Len() > 0 {
			lv, _ := rtts.Latest()
			wait = time.Duration(lv.(LogDuration)) / 5
		} else {
			wait = time.Duration(5) * time.Millisecond
		}
		<-time.After(wait)

		rin := ThroughputFromTimes(sendTimes)

		throughputLock.Lock()

		rout := throughput
		zt := est_bandwidth*(rin/rout) - rin
		lv, err := rtts.Latest()
		if err != nil {
			continue
		}

		rtt := time.Duration(lv.(LogDuration))

		flowRateLock.Lock()

		if !xtcpData.xtcp_mode {
			flowRate = updateRateDelay(flowRate, est_bandwidth, rin, zt, rtt)
		} else {
			flowRate = updateRateXtcp(rtt)
		}

		if flowRate < 0 {
			panic("negative flow rate")
		}

		flowRateLock.Unlock()
		throughputLock.Unlock()
	}
}

// read the current flow rate and set the pacing channel appropriately
func flowPacer(pacing chan interface{}) {
	for {
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
) error {
	pacing := make(chan interface{})
	go flowPacer(pacing)

	for {
		pkt := Packet{
			SeqNo:   xtcpData.seq_nos[xtcpData.currVirtFlow],
			VirtFid: xtcpData.currVirtFlow,
			Rtt:     Now(),
			Payload: "",
		}

		//fmt.Println("sending ", Now(), pkt.VirtFid, pkt.SeqNo)

		incrementXtcpSeq()

		sendTimes.Add(intLogVal(Now()))
		err := SendPacket(conn, pkt, 1480)
		if err != nil {
			fmt.Println(err)
		}

		<-pacing
	}
}

func handleAck(
	conn *net.UDPConn,
	expSrc *net.UDPAddr,
	rtt_history chan int64,
	throughput_history chan float64,
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

		ackTimes.Add(intLogVal(Now()))
		rout := ThroughputFromTimes(ackTimes)

		// check for XTCP packet drop
		if drop := checkXtcpSeq(pkt.VirtFid, pkt.SeqNo); !drop {
			fmt.Println("drop", pkt.VirtFid, pkt.SeqNo-1)
			//dropDetected(pkt.VirtFid) // TODO enable switching. for now only delay mode
		} else {
			increaseXtcpWind(pkt.VirtFid)
		}

		rtt_history <- pkt.Rtt * 2 // multiply one-way delay by 2
		throughput_history <- rout
	}
}
