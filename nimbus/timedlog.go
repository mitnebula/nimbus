package main

import (
	"fmt"
	"sync"
	"time"
)

type TimedLog struct {
	length time.Duration        // constraint on newest time - oldest time
	m      map[Packet]time.Time // seqno -> time
	t      map[time.Time]Packet // time -> seqno
	times  []time.Time          // sorted slice of keys in map
	mux    sync.Mutex           // for thread-safeness
}

func InitTimedLog(d time.Duration) *TimedLog {
	return &TimedLog{
		length: d,
		m:      make(map[Packet]time.Time),
		t:      make(map[time.Time]Packet),
		times:  make([]time.Time, 0),
	}
}

func (l *TimedLog) UpdateDuration(d time.Duration) {
	l.length = d
}

func (l *TimedLog) Len() int {
	return len(l.times)
}

func (l *TimedLog) Add(t time.Time, pkt Packet) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.times = append(l.times, t)
	l.m[pkt] = t
	l.t[t] = pkt

	if len(l.times) != len(l.m) {
		panic("TimedLog in inconsistent state")
	}

	currSpan := l.times[len(l.times)-1].Sub(l.times[0])
	// remove older, keep at least 100
	for len(l.times) > 100 && currSpan > l.length {
		rem := l.times[0]
		seq, _ := l.t[rem] // seq
		delete(l.t, rem)
		delete(l.m, seq)
		l.times = l.times[1:]
	}
}

// last packet before given time and time it was logged
func (l *TimedLog) Before(wanted time.Time) (Packet, time.Time, error) {
	var then time.Time

	if wanted.Before(l.times[0]) {
		return l.t[l.times[0]], l.times[0], fmt.Errorf("wanted time before log start")
	}

	for _, t := range l.times {
		if t.After(wanted) {
			return l.t[then], then, nil
		} else {
			then = t
		}
	}

	lastTime := l.times[len(l.times)-1]
	return l.t[lastTime], lastTime, nil
}

func (l *TimedLog) NumPacketsBetween(start time.Time, end time.Time) (int, error) {
	count := 0
	for _, t := range l.times {
		if !t.Before(end) {
			return count, nil
		} else if !t.Before(start) {
			count++
		}
	}

	return count, nil
}
