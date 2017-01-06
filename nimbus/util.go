package main

import (
	"time"
)

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

func ThroughputFromTimes(times *TimedLog, delay time.Duration) (float64, error) {
	//double rout = ((ack_times.size() - 1) * 1490 * 8.0) / (now - this->ack_times.front().second);
	newest, _, err := times.Latest(delay)
	if err != nil {
		return 0, err
	}

	oldest, _, err := times.Oldest(delay)
	if err != nil {
		return 0, err
	}

	dur := time.Unix(0, int64(newest.(intLogVal))).Sub(time.Unix(0, int64(oldest.(intLogVal))))

	return float64((times.Len()-1)*1480*8.0) / dur.Seconds(), nil
}

func MinRtt(rtts *Log) time.Duration {
	var min_rtt time.Duration
	lv, err := rtts.Min()
	if err != nil {
		min_rtt, _ = time.ParseDuration("0s")
	}
	min_rtt = time.Duration(lv.(durationLogVal))
	return min_rtt
}
