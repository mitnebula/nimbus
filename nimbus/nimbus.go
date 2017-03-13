package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"

	"github.mit.edu/hari/nimbus-cc/receiver"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var ip = flag.String("ip", "127.0.0.1", "IP to connect to")
var port = flag.String("port", "42424", "Port to connect to/listen on")
var mode = flag.String("mode", "client", "server|client|sender|receiver")
var runtime = flag.Duration("t", time.Duration(180)*time.Second, "runtime as duration")
var initLogLevel = flag.String("log", "info", "logrus log level: (info | warn | debug | error | fatal | panic)")

var numVirtualFlows = flag.Int("numFlows", 1, "number of virtual flows")
var estBandwidth = flag.Float64("estBandwidth", 24e6, "estimated bandwidth")
var pulseSize = flag.Float64("pulseSize", 0.5, "size of pulses to send as fraction of rate")
var initUseSwitching = flag.Bool("useSwitching", true, "if false, do not pulse, always stay in delay mode")
var initMode = flag.String("initMode", "DELAY", "mode to start in: (DELAY | XTCP)")
var initReportInterval = flag.Duration("reportInterval", time.Duration(2)*time.Second, "how often to report throughput and delay, duration")
var initDelayThreshold = flag.Float64("delayThreshold", 1.25, "use delay threshold of min_rtt * X")

// TODO make a slow start-like startup
var initRate = flag.Float64("initRate", 10e6, "initial sending rate")
var measurementTimescale = flag.Int64("measurementTimescale", 1, "over how many rtts to measure rin, rout, zt, elasticity")
var measurementInterval = flag.Duration("measurementInterval", time.Duration(10)*time.Millisecond, "how often to measure rin, rout, zt, elasticity")

// overall statistics
var done chan interface{}
var sendCount int64
var recvCount int64
var startTime time.Time
var endTime time.Time

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	flag.Parse()

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "15:04:05.000000"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	lvl, parseErr := log.ParseLevel(*initLogLevel)
	if parseErr != nil {
		log.Fatal(parseErr)
	}
	log.SetLevel(lvl)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Error("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Error("could not start CPU profile: ", err)
		}
		log.WithFields(log.Fields{
			"Starting Profiling, Profile File":              *cpuprofile,
		}).Info("Starting ", *mode)
		defer pprof.StopCPUProfile()
	}

	log.WithFields(log.Fields{
		"log":              *initLogLevel,
		"ip":               *ip,
		"port":             *port,
		"runtime":          *runtime,
		"switching":        *initUseSwitching,
		"initMode":         *initMode,
		"interval":         *initReportInterval,
		"delayThreshold":   *initDelayThreshold,
		"numFlows":         *numVirtualFlows,
		"est_bandwidth":    *estBandwidth,
		"pulseSize":        *pulseSize,
		"measureTimescale": *measurementTimescale,
		"measureInterval":  *measurementInterval,
	}).Info("Starting ", *mode)

	//print on ctrl-c
	done = make(chan interface{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go exitStats(interrupt)

	currMode = *initMode
	switch *initMode {
	case "DELAY":
		flowMode = DELAY
	case "XTCP":
		flowMode = XTCP
	default:
		log.Panicf("Unknown initial modea %s!", *initMode)
	}

	est_bandwidth = *estBandwidth
	flowRate = *initRate
	reportInterval = *initReportInterval
	useSwitching = *initUseSwitching
	delayThreshold = *initDelayThreshold

	xtcpData.numVirtualFlows = uint16(*numVirtualFlows)
	xtcpData.setXtcpCwnd(flowRate, time.Duration(150)*time.Millisecond)

	startTime = time.Now()

	var err error
	if *mode == "server" {
		err = Server(*port)

		endTime = startTime.Add(*runtime)
	} else if *mode == "sender" {
		err = Sender(*ip, *port)

		endTime = startTime.Add(*runtime)
	} else if *mode == "client" || *mode == "receiver" {
		syn, cnt, off := setupReceiver()

		if *mode == "client" {
			err = receiver.Client(*ip, *port, syn, cnt, off)
		} else {
			err = receiver.Receiver(*port, syn, cnt, off)
		}

		endTime = startTime.Add(*runtime)
	}
	if err != nil {
		log.Error(err)
	}

	<-done
}

func exitStats(interrupt chan os.Signal) {
	<-interrupt
	doExit()
}

func doExit() {
	elapsed := time.Since(startTime)
	sendBytes := float64(sendCount * ONE_PACKET)
	recvBytes := float64(recvCount * ONE_PACKET)
	log.WithFields(log.Fields{
		"throughput": BpsToMbps(sendBytes / elapsed.Seconds()),
		"pkts":       sendCount,
	}).Info("Send statistics")
	log.WithFields(log.Fields{
		"throughput": BpsToMbps(recvBytes / elapsed.Seconds()),
		"pkts":       recvCount,
	}).Info("Receive statistics")
	log.Info(elapsed, " elapsed")
	done <- struct{}{}
	//os.Exit(0)

}
