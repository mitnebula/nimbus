package main

import (
	log "github.com/Sirupsen/logrus"
	"math"
	"time"
	fft "github.com/mjibson/go-dsp/fft"
	"github.com/akshayknarayan/history"
	"math/cmplx"
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
var zout_history *history.History

var modeSwitchTime time.Time

// test state
var maxQd time.Duration

var untilNextUpdate time.Duration

var currMode string

func init() {
	est_bandwidth = 10e6

	zt_history = history.MakeHistory(min_rtt)
	xt_history = history.MakeHistory(min_rtt)
	zout_history = history.MakeHistory(min_rtt)
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
	} else if zt>est_bandwidth {
		zt = est_bandwidth
	}

	if zt > 0 {
		zt_history.Add(time.Now(), zt)
	}
	xt_history.Add(time.Now(), rtt)
	zout_history.Add(time.Now(), est_bandwidth-rout)	

	log.WithFields(log.Fields{
		"elapsed":  time.Since(startTime),
		"zt":       zt,
		"rtt":      rtt,
		"rin":      rin,
		"rout":     rout,
		"flowRate": flowRate,
	}).Debug()
	return
}

func changePulses(fr float64) float64 {
	elapsed := time.Since(startTime).Seconds()
	return fr + (*pulseSize)*fr*math.Sin((1/(2*min_rtt.Seconds()))*2*math.Pi*elapsed)
}

func shouldSwitch (rtt time.Duration){
	// not implemented
	// TODO FFT-based switching logic
	
	//Too short a duration don't switch 
	if time.Since(startTime) < 15*time.Second {
	return
	}

	// gather history and correct for phase shifts

 	duration_of_fft := 5.0
	thresh := 0.5
	end_time_snapshot := time.Now()
	start_time_snapshot := time.Now().Add(-time.Duration(duration_of_fft+1)*time.Second)
	
	raw_zt, _ := zt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)  
	raw_rtt, _ := xt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	raw_zout, _ := zout_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	
	clean_zt := []float64{}
	clean_zout := []float64{}
	
	T := 0.01
	N := int(duration_of_fft/T)
	
	for i := 0; i<N; i++ {
		clean_zt = append(clean_zt, raw_zt[i+int(raw_rtt[i].Item.(time.Duration).Seconds()/T)].Item.(float64))
		clean_zout = append(clean_zout, raw_zout[i].Item.(float64))
	}
	// TODO add hanning and detrending for FFTs 
	
	detrend(clean_zt)	
	detrend(clean_zout)	
		
	fft_zt := fft.FFTReal(clean_zt)
	fft_zout := fft.FFTReal(clean_zout)

 
	freq := []float64{}	
	for i := 0; i<N/2; i++ {
		freq = append(freq, float64(i)*(1.0/(float64(N)*T)))
	}
	
	zout_peak := findPeak(2.0, 10.0, freq, fft_zout)  
	zt_peak := findPeak(2.0, 10.0, freq, fft_zt)
	

	// These conditions depend on where we want out peak to be
	if 4.5<freq[zout_peak] && freq[zout_peak]<5.5 {
	 	if 4.5<freq[zt_peak] && freq[zt_peak]<5.5 {
			if cmplx.Abs(fft_zt[zt_peak])>thresh*cmplx.Abs(fft_zout[zout_peak]) {
				switchToXtcp(rtt)
			} else if  cmplx.Abs(fft_zt[zt_peak])<0.6*thresh*cmplx.Abs(fft_zout[zout_peak]) {
				switchToDelay(rtt)			
			}
		} else {
			switchToDelay(rtt)		
		}
	}

	log.WithFields(log.Fields{
		"elapsed":  time.Since(startTime),
		"ZoutPeak":       freq[zout_peak],
		"ZtPeak":      freq[zt_peak],
		"ZoutPeakVal":      cmplx.Abs(fft_zout[zout_peak]),
		"ZtPeakVal":     cmplx.Abs(fft_zt[zt_peak]),
	}).Debug()	

	return
}

func findPeak(start_freq float64, end_freq float64,xf []float64, fft []complex128) int {
	max_ind := 0			
	for j:=0; j<(len(xf)) ; j++ {
		if xf[j]<start_freq {
			max_ind=j
			continue
		}
		if xf[j]>end_freq{
			break
		}
		if cmplx.Abs(fft[j])>cmplx.Abs(fft[max_ind]){
			max_ind=j
		}	
	}
	return max_ind
}
func mean(a []float64) float64{
	mean_val := 0.0
	for i:=0;i<len(a);i++ {
		mean_val+=a[i]
	}
	return mean_val/float64(len(a))
}
func detrend(a []float64) {
	mean_val := mean(a)
	for i:=0;i<len(a);i++ {
		a[i]-=mean_val
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
