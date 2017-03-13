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
	if zt < 0 {
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

	measurementWindow := (4.8e6/est_bandwidth)	
	phase := elapsed/(2*measurementWindow)
	phase -= math.Floor(phase)	
	upRatio := (min_rtt.Seconds())/(2*measurementWindow)
	if phase<upRatio {
		return fr + (*pulseSize)*fr_modified*math.Sin(2*math.Pi*phase*(0.5/upRatio))
	} else {
		return max(0.05*est_bandwidth, fr + (upRatio/(1.0-upRatio))*(*pulseSize)*fr_modified*math.Sin(2*math.Pi*(0.5 + (phase-upRatio)*(0.5/(1.0-upRatio)))))
	}

}

func shouldSwitch (rtt time.Duration){
	

	// gather history and correct for phase shifts

	measurementWindow := (4.8e6/est_bandwidth)

 	duration_of_fft := 50*measurementWindow
	
	//Too short a duration don't switch 
	if time.Since(startTime) < time.Duration(10.0 + 1.0*duration_of_fft)*time.Second {
	return
	}
		
	thresh := 0.5
	end_time_snapshot := time.Now()
	start_time_snapshot := time.Now().Add(-time.Duration(duration_of_fft+1.0)*time.Second)
	
	raw_zt, _ := zt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)  
	raw_rtt, _ := xt_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	raw_zout, _ := zout_history.ItemsBetween(start_time_snapshot, end_time_snapshot)
	
	


	clean_zt := []float64{}
	clean_zout := []float64{}
	//N must be a power of 2
	T := measurementInterval.Seconds()
	N := int(duration_of_fft/T)
	for i:=1 ; ;i*=2 {
		if i>=N {
			N=i
			break
		}
	}

	//correct for missing entries
	start := start_time_snapshot
	/*corrected_rtt := make([]history.HistoryItemWithTime, 0)
	corrected_zt := make([]history.HistoryItemWithTime, 0)
	corrected_zout := make([]history.HistoryItemWithTime, 0)
	for j:=0; j<len(raw_zt); {
		for ;j<len(raw_zt) && raw_zt[j].Time.Before(start); {
			j += 1 
		}
		if j>=len(raw_zt) {
			break
		}
		corrected_rtt = append(corrected_rtt, raw_rtt[j])
		corrected_zt = append(corrected_zt, raw_zt[j])
		corrected_zout = append(corrected_zout, raw_zout[j])
		start = start.Add(*measurementInterval)
	}*/


	//Correct got Phase Shifts
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
	}

	if mean(clean_zt)<0.3*est_bandwidth {
		switchToDelay(rtt)
		return
	}

	// TODO add hanning
	
	detrend(clean_zt)	
	detrend(clean_zout)	
	start = time.Now()	
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
	zout_peak := findPeak(0.8*expected_peak, 1.6*expected_peak, freq, fft_zout)  
	zt_peak := findPeak(0.8*expected_peak, 1.6*expected_peak, freq, fft_zt)	

	if expected_peak-0.5<freq[zout_peak] && freq[zout_peak]<expected_peak+0.5 {
	 	if expected_peak-0.5<freq[zt_peak] && freq[zt_peak]<expected_peak+0.5 {
			if cmplx.Abs(fft_zt[zt_peak])>thresh*cmplx.Abs(fft_zout[zout_peak]) {
				switchToXtcp(rtt)
			} else if  cmplx.Abs(fft_zt[zt_peak])<0.5*thresh*cmplx.Abs(fft_zout[zout_peak]) {
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
		"Expected Peak":   expected_peak,
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

func max(a, b float64) float64{
	if a > b {
		return a 
	} else {
		return b
	}
}

func doUpdate() {
	

	//TODO: measurement window shouldn't be less than RTT
	measurementWindow := int64(1.0e9 * (4.8e6/est_bandwidth))

	rin, _, zt, rtt, err := measure(time.Duration(*measurementTimescale*measurementWindow) * time.Nanosecond)
	if err != nil {
		log.WithFields(log.Fields{
			"elapsed":  time.Since(startTime),
			"error":    err,
			}).Debug()
		return
	}
	start := time.Now()
	shouldSwitch(rtt)
	end := time.Now()

	if end.Sub(start) > 5*time.Millisecond {
		log.WithFields(log.Fields{
			"elapsed":  time.Since(startTime),
			"ShouldSwitch Time": end.Sub(start).Seconds(),
		}).Debug()
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


	flowRate = changePulses(flowRate)

	if flowRate < 0 {
		log.Panic("negative flow rate")
	}
}

func flowRateUpdater() {
	//TODO: measurement interval should kind of depend on duration of FFT
	for _ = range time.Tick(*measurementInterval) {
		doUpdate()

		if time.Now().After(endTime) {
			doExit()
		}
	}
}
