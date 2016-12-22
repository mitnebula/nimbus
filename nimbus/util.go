package main

import (
	"time"
)

type LogDuration time.Duration

func (d LogDuration) lessthan(o LogVal) bool {
	return time.Duration(d) < time.Duration(o.(LogDuration))
}

func MakeBytes(size int) string {
	result := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		result = append(result, 'a')
	}

	return string(result)
}

func Now() int64 {
	return time.Now().UnixNano()
}

func ThroughputFromTimes(times *Log) float64 {
	//double rout = ((ack_times.size() - 1) * 1490 * 8.0) / (now - this->ack_times.front().second);
	oldest, newest, err := times.Ends()
	if err != nil {
		return 0
	}

	dur := time.Unix(0, int64(newest.(intLogVal))).Sub(time.Unix(0, int64(oldest.(intLogVal))))

	return float64((times.Len()-1)*1480*8.0) / dur.Seconds()
}

func MinRtt(rtts *Log) time.Duration {
	var min_rtt time.Duration
	lv, err := rtts.Min()
	if err != nil {
		min_rtt, _ = time.ParseDuration("0s")
	}
	min_rtt = time.Duration(lv.(LogDuration))
	return min_rtt
}
