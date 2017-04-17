package main

import (
	log "github.com/Sirupsen/logrus"
	"math"
	"time"
	fft "github.com/mjibson/go-dsp/fft"
	win "github.com/mjibson/go-dsp/window"
	"github.com/akshayknarayan/history"
	"math/cmplx"
)

const (
	alpha = 0.8
	switchallowed = true
	thresh = 0.4
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
//TODO make flowrate numbers avg
func switchToDelay(rtt time.Duration) {
	if !switchallowed {
		return
	}
	if flowMode == DELAY {
		return
	}

	delayThreshold = max(*initDelayThreshold, rtt.Seconds()/min_rtt.Seconds())
	log.WithFields(log.Fields{
		"elapsed": time.Since(startTime),
		"from":    currMode,
		"to":      "DELAY",
		"DelayTheshold":  delayThreshold,
	}).Info("switched mode")

	flowMode = DELAY
	currMode = "DELAY"
	modeSwitchTime = time.Now()
}
//TODO make flowrate numbers avg
func switchToXtcp(rtt time.Duration) {
	if !switchallowed {
		return
	}
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
	
	//gradually bringdown target delay, 2% everymin_rtt
	if delayThreshold>*initDelayThreshold{
		delayThreshold -= (measurementInterval.Seconds()/0.1)*0.02
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
	if math.IsNaN(zt) || zt < 0{
		zt = 0
	} else if zt>est_bandwidth {
		zt = est_bandwidth
	}

	//if zt > 0 {
		zt_history.Add(time.Now(), zt)
	//}
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
	
	fr_modified := est_bandwidth

	measurementWindow := math.Max((4.8e6/est_bandwidth),min_rtt.Seconds())	
	phase := elapsed/(2*measurementWindow)
	phase -= math.Floor(phase)	
	upRatio := 0.25
	if phase<upRatio {
		return fr + (*pulseSize)*fr_modified*math.Sin(2*math.Pi*phase*(0.5/upRatio))
	} else {
		return max(0.05*est_bandwidth, fr + (upRatio/(1.0-upRatio))*(*pulseSize)*fr_modified*math.Sin(2*math.Pi*(0.5 + (phase-upRatio)*(0.5/(1.0-upRatio)))))
	}

}

func shouldSwitch (rtt time.Duration){

	measurementWindow := math.Max((4.8e6/est_bandwidth),min_rtt.Seconds())

 	duration_of_fft := 50*measurementWindow
	
	//Too short a duration don't switch 
	if time.Since(startTime) < time.Duration(10.0 + 1.0*duration_of_fft)*time.Second {
	return
	}
		
	
	end_time_snapshot := time.Now()
	start_time_snapshot := time.Now().Add(-time.Duration(duration_of_fft+1.0)*time.Second)
	
	raw_zt, _ := zt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)  
	raw_rtt, _ := xt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	raw_zout, _ := zout_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	
	


	clean_zt := []float64{}
	clean_zout := []float64{}
	clean_rtt := []float64{}
	//N must be a power of 2
	T := measurementInterval.Seconds()
	N := int(duration_of_fft/T)
	for i:=1 ; ;i*=2 {
		if i>=N {
			N=i
			break
		}
	}

	for i := 0; i<N; i++ {
		if i>=len(raw_rtt) {
			return
		}
		j := i+int(raw_rtt[i].Item.(time.Duration).Seconds()/T)
		if j>=len(raw_zt) {
			return
		}
		clean_zt = append(clean_zt, raw_zt[j].Item.(float64))
		clean_zout = append(clean_zout, raw_zout[i].Item.(float64))
		clean_rtt = append(clean_rtt, raw_rtt[i].Item.(time.Duration).Seconds())
	}

	avg_rtt := time.Duration(1000*mean(clean_rtt))*time.Millisecond

	switched_already := false
	if mean(clean_zt)<0.2*est_bandwidth {
		switchToDelay(avg_rtt)
		switched_already=true
	} else if mean(clean_zt)>0.8*est_bandwidth {
		switchToXtcp(avg_rtt)
		switched_already=true
	}
	
	detrend(clean_zt)	
	detrend(clean_zout)	
	start := time.Now()
	hann_window := win.Hann(len(clean_zt))
	for i:=0; i<len(clean_zt); i++ {
		clean_zt[i] = clean_zt[i]*hann_window[i]
		clean_zout[i] = clean_zout[i]*hann_window[i]
	}
	fft_zt := fft.FFTReal(clean_zt)
	fft_zout := fft.FFTReal(clean_zout)
	end := time.Now()	
	if end.Sub(start) > 5*time.Millisecond {
		log.WithFields(log.Fields{
			"elapsed":  time.Since(startTime),
			"FFT Time": end.Sub(start).Seconds(),
		}).Debug()
	}
 
	freq := []float64{}	
	for i := 0; i<N/2; i++ {
		freq = append(freq, float64(i)*(1.0/(float64(N)*T)))
	}

	//Pluse Size is fixed to 2*measurementWindow
	expected_peak := 1.0/(2*measurementWindow)
	_, mean_zt:= findPeak(1.2*expected_peak, 1.6*expected_peak, freq, fft_zt)	
	exp_peak_zt, _ := findPeak(0.9*expected_peak, 1.1*expected_peak, freq, fft_zt)
	exp_peak_zout, _ := findPeak(0.9*expected_peak, 1.1*expected_peak, freq, fft_zout)

	elasticity := (cmplx.Abs(fft_zt[exp_peak_zt])-mean_zt)/(cmplx.Abs(fft_zout[exp_peak_zout]))
	if elasticity>thresh*1 && !switched_already{
		switchToXtcp(avg_rtt)
	} else if elasticity<thresh*0.75 && !switched_already{
		switchToDelay(avg_rtt)
	}
	log.WithFields(log.Fields{
		"elapsed":  time.Since(startTime),
		"ZoutPeakVal":      cmplx.Abs(fft_zout[exp_peak_zout]),
		"ZtPeakVal":     cmplx.Abs(fft_zt[exp_peak_zt]),
		"Expected Peak":   expected_peak,
		"Elasticity": elasticity,
	}).Debug()	

	return
}

func findPeak(start_freq float64, end_freq float64,xf []float64, fft []complex128) (int, float64) {
	max_ind := 0
	mean := 0.0
	count := 0.0
	for j:=0; j<(len(xf)) ; j++ {
		if xf[j]<=start_freq {
			max_ind=j
			continue
		}
		if xf[j]>end_freq{
			break
		}
		mean += cmplx.Abs(fft[j])
		count += 1.0
		if cmplx.Abs(fft[j])>cmplx.Abs(fft[max_ind]){
			max_ind=j
		}	
	}
	return max_ind, mean/max(count, 1.0)
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

func max(a, b float64) float64{
	if a > b {
		return a 
	} else {
		return b
	}
}

func doUpdate() {

	measurementWindow := int64(1.0e9 * math.Max((4.8e6/est_bandwidth),min_rtt.Seconds()))

	rin, _, zt, rtt, err := measure(time.Duration(*measurementTimescale*measurementWindow) * time.Nanosecond)
	if err != nil {
		log.WithFields(log.Fields{
			"elapsed":  time.Since(startTime),
			"error":    err,
			}).Debug()
		return
	}

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

	shouldSwitch(rtt)

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