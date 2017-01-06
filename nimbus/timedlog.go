package main

import (
	"fmt"
	"sync"
	"time"
)

type TimedLog struct {
	length time.Duration        // constraint on newest time - oldest time
	m      map[time.Time]LogVal // the map
	times  []time.Time          // sorted slice of keys in map
	mux    sync.Mutex           // for thread-safeness
}

func InitTimedLog(d time.Duration) *TimedLog {
	return &TimedLog{length: d, m: make(map[time.Time]LogVal), times: make([]time.Time, 0)}
}

func CurrSpan(arr []time.Time) time.Duration {
	if len(arr) < 1 {
		return time.Duration(0)
	}

	oldest := arr[0]
	newest := arr[len(arr)-1]
	return newest.Sub(oldest)
}

func (l *TimedLog) UpdateDuration(d time.Duration) {
	l.length = d
}

func (l *TimedLog) Len() int {
	return len(l.times)
}

func (l *TimedLog) Add(t time.Time, v LogVal) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if len(l.times) > 0 && t.Before(l.times[len(l.times)-1]) {
		panic("times list unsorted")
	}

	l.times = append(l.times, t)
	l.m[t] = v

	if len(l.times) != len(l.m) {
		panic("TimedLog in inconsistent state")
	}

	// remove older, keep at least 100
	for len(l.times) > 100 && CurrSpan(l.times) > l.length {
		rem := l.times[0]
		delete(l.m, rem)
		l.times = l.times[1:]
		if rem.After(l.times[0]) {
			panic("times list unsorted")
		}
	}
}

func (l *TimedLog) Min() (LogVal, time.Time, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if len(l.times) == 0 {
		return intLogVal(0), time.Now(), fmt.Errorf("empty log")
	}

	least := l.m[l.times[0]]
	then := l.times[0]
	for _, t := range l.times {
		if l.m[t].lessthan(least) {
			least = l.m[t]
			then = t
		}
	}

	return least, then, nil
}

func (l *TimedLog) Avg() (LogVal, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	n := len(l.m)
	if n == 0 {
		return intLogVal(0), fmt.Errorf("empty log")
	}

	var sum LogVal
	var dv LogVal

	switch l.m[l.times[0]].(type) {
	case intLogVal:
		sum = intLogVal(0)
		dv = intLogVal(n)
	case floatLogVal:
		sum = floatLogVal(0)
		dv = floatLogVal(float64(n))
	case durationLogVal:
		sum = durationLogVal(time.Duration(0))
		dv = durationLogVal(time.Duration(n))
	}

	for _, v := range l.m {
		sum = sum.add(v)
	}

	return sum.div(dv), nil
}

func (l *TimedLog) CountBetween(start time.Time, end time.Time) (int, error) {
	if len(l.times) == 0 {
		return 0, fmt.Errorf("empty log")
	}

	count := 0

	for _, t := range l.times {
		switch {
		case t.Equal(start) || t.After(start):
			fallthrough
		case t.Equal(end) || t.Before(end):
			count++
		case t.After(end):
			return count, nil
		}
	}

	return count, nil
}

// last value before given time
func (l *TimedLog) Before(wanted time.Time) (LogVal, time.Time, error) {
	if len(l.times) == 0 {
		return intLogVal(0), time.Now(), fmt.Errorf("empty log")
	}

	if wanted.Before(l.times[0]) {
		return intLogVal(0), time.Now(), fmt.Errorf("before: time not covered")
	}

	lastTime := l.times[0]
	for _, t := range l.times {
		if t.Equal(wanted) || t.Before(wanted) {
			lastTime = t
		} else {
			break
		}
	}

	return l.m[lastTime], lastTime, nil
}

// first value after given time
func (l *TimedLog) After(wanted time.Time) (LogVal, time.Time, error) {
	if len(l.times) == 0 {
		return intLogVal(0), time.Now(), fmt.Errorf("empty log")
	}

	for _, t := range l.times {
		if t.Equal(wanted) || t.After(wanted) {
			return l.m[t], t, nil
		}
	}

	return intLogVal(0), time.Now(), fmt.Errorf("after: time not covered %v %v", wanted, l.times[len(l.times)-1])
}
