package main

import (
	"fmt"
	//"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	ONE_PACKET = 1500.0 * 8.0
)

var flowRate float64
var flowRateLock sync.Mutex

var min_rtt time.Duration

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

// overall statistics
var sendCount int64
var recvCount int64
var startTime time.Time

func init() {
	flowMode = DELAY

	flowRate = 1e6
	min_rtt = time.Duration(999) * time.Hour

	// (est_bandwidth / min_rtt) * C where 0 < C < 1, use C = 0.4
	beta = (flowRate / 0.001) * 0.33

	rtts = InitLog(900)
	sendTimes = InitTimedLog(min_rtt)
	ackTimes = InitTimedLog(min_rtt)
	rin_history = InitLog(500)

	sendCount = 0
	recvCount = 0
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
	go rttUpdater(rtt_history)

	err = synAckExchange(conn, addr, rtt_history)
	if err != nil {
		return err
	}

	fmt.Println("connected")

	//print on ctrl-c
	procExit := make(chan os.Signal, 1)
	signal.Notify(procExit, os.Interrupt)

	go handleAck(conn, addr, rtt_history, recvExit)
	go flowRateUpdater()
	go output()

	startTime = time.Now()
	go send(conn, recvExit)
	go exitStats(procExit, recvExit)

	<-recvExit

	return nil
}

func synAckExchange(conn *net.UDPConn, expSrc *net.UDPAddr, rtt_history chan int64) error {
	seq, vfid := xtcpData.getNextSeq()
	syn := Packet{
		SeqNo:   seq,
		VirtFid: vfid,
		Echo:    Now(),
		Payload: "SYN",
	}

	err := SendAck(conn, syn)
	if err != nil {
		return err
	}

	ack, srcAddr, err := RecvPacket(conn)
	if err != nil {
		return err
	}
	if srcAddr.String() != expSrc.String() {
		return fmt.Errorf("got packet from unexpected src: %s; expected %s", srcAddr, expSrc)
	}

	xtcpData.checkXtcpSeq(ack.VirtFid, ack.SeqNo)

	delay := ack.RecvTime - ack.Echo // one way delay
	rtt_history <- delay * 2         // multiply one-way delay by 2

	return nil
}

func exitStats(procExit chan os.Signal, done chan interface{}) {
	<-procExit
	elapsed := time.Since(startTime)
	totalBytes := float64(sendCount * ONE_PACKET)
	fmt.Printf("Sent: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), sendCount, elapsed)
	totalBytes = float64(recvCount * ONE_PACKET)
	fmt.Printf("Received: throughput %.4v; %v packets in %v\n", totalBytes/elapsed.Seconds(), recvCount, elapsed)
	done <- struct{}{}
}

func output() {
	for _ = range time.Tick(2 * time.Second) {
		rtt, _ := rtts.Latest()
		inTpt, _, _, err := ThroughputFromTimes(sendTimes, time.Now(), time.Duration(2)*time.Second)
		if err != nil {
			inTpt = 0
		}
		outTpt, _, _, err := ThroughputFromTimes(ackTimes, time.Now(), time.Duration(2)*time.Second)
		if err != nil {
			outTpt = 0
		}
		fmt.Printf("%v : %v %v %v %v\n", Now(), inTpt, outTpt, time.Duration(rtt.(durationLogVal)), min_rtt)
	}
}

// keep rtt up to date (from received acks)
func rttUpdater(rtt_history chan int64) {
	for t := range rtt_history {
		rtt := time.Duration(t) * time.Nanosecond
		if rtt < min_rtt {
			min_rtt = rtt

			sendTimes.UpdateDuration(rtt * 100)
			ackTimes.UpdateDuration(rtt * 100)
		}

		rtts.Add(durationLogVal(rtt))
	}
}

// read the current flow rate and set the pacing channel appropriately
func flowPacer(pacing chan interface{}) {
	credit := float64(ONE_PACKET)
	maxBurst := int64(0)
	sumBurst := int64(0)
	countBurst := int64(0)
	lastTime := time.Now()
	//lastStatTime := lastTime

	for _ = range time.Tick(time.Duration(100) * time.Microsecond) {
		elapsed := time.Since(lastTime)
		lastTime = time.Now()
		credit += elapsed.Seconds() * flowRate
		if credit > 100*ONE_PACKET {
			credit = 100 * ONE_PACKET
		}

		burst := int64(credit / ONE_PACKET)
		sumBurst = sumBurst + burst
		countBurst = countBurst + 1
		if burst > maxBurst {
			maxBurst = burst
		}
		//if time.Since(lastStatTime) > time.Duration(1)*time.Second {
		//	fmt.Printf("Current burst = %d, Avg burst = %d, Max burst = %d pkts\n", burst, sumBurst/countBurst, maxBurst)
		//	lastStatTime = lastTime
		//}

		for credit >= ONE_PACKET {
			pacing <- struct{}{}
			credit -= ONE_PACKET
		}
	}
}

func send(
	conn *net.UDPConn,
	done chan interface{},
) error {
	pacing := make(chan interface{})
	go flowPacer(pacing)

	//lastSend := time.Now()
	for {
		seq, vfid := xtcpData.getNextSeq()

		pkt := Packet{
			SeqNo:   seq,
			VirtFid: vfid,
			Echo:    Now(),
			Payload: "",
		}

		sendTimes.Add(time.Now(), pkt)
		err := SendPacket(conn, pkt, 1500)
		if err != nil {
			fmt.Println(err)
		}

		//fmt.Printf("%v sending (%v, %v)\n", time.Now(), seq, vfid)
		//lastSend = time.Now()

		sendCount++
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
		if srcAddr.String() != expSrc.String() {
			fmt.Println(fmt.Errorf("got packet from unexpected src: %s; expected %s", srcAddr, expSrc))
		}

		recvTime := time.Unix(0, pkt.RecvTime)
		ackTimes.Add(recvTime, pkt)
		recvCount++

		// check for XTCP packet drop
		if ok, exp := xtcpData.checkXtcpSeq(pkt.VirtFid, pkt.SeqNo); !ok {
			err := fmt.Errorf("drop %v %v %v %v", pkt.VirtFid, pkt.SeqNo, exp, recvCount)
			fmt.Println(err)
			done <- struct{}{}
			break
			//xtcpData.dropDetected(pkt.VirtFid)
		} else {
			xtcpData.increaseXtcpWind(pkt.VirtFid)
		}

		delay := pkt.RecvTime - pkt.Echo // one way delay

		//fmt.Printf("%v recv ack (%v,%v) rt %v s_ech %v rtt %v\n", time.Now(), pkt.VirtFid, pkt.SeqNo, pkt.RecvTime, pkt.Echo, delay*2)
		rtt_history <- delay * 2 // multiply one-way delay by 2
	}
}
