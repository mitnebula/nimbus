package main

import (
	"fmt"
	"sync"
	"time"
)

type LogVal interface {
	lessthan(o LogVal) bool
	add(o LogVal) LogVal
	div(o LogVal) LogVal
}

type intLogVal int64

func (i intLogVal) lessthan(o LogVal) bool {
	return int64(i) < int64(o.(intLogVal))
}

func (i intLogVal) add(o LogVal) LogVal {
	return intLogVal(int64(i) + int64(o.(intLogVal)))
}

func (i intLogVal) div(o LogVal) LogVal {
	return intLogVal(int64(i) / int64(o.(intLogVal)))
}

type floatLogVal float64

func (f floatLogVal) lessthan(o LogVal) bool {
	return float64(f) < float64(o.(floatLogVal))
}

func (f floatLogVal) add(o LogVal) LogVal {
	return floatLogVal(float64(f) + float64(o.(floatLogVal)))
}

func (f floatLogVal) div(o LogVal) LogVal {
	return floatLogVal(float64(f) / float64(o.(floatLogVal)))
}

type durationLogVal time.Duration

func (d durationLogVal) lessthan(o LogVal) bool {
	return time.Duration(d) < time.Duration(o.(durationLogVal))
}

func (d durationLogVal) add(o LogVal) LogVal {
	return durationLogVal(time.Duration(d) + time.Duration(o.(durationLogVal)))
}

func (d durationLogVal) div(o LogVal) LogVal {
	ns := time.Duration(d).Nanoseconds()
	dv := time.Duration(o.(durationLogVal)).Nanoseconds()

	res := ns / dv

	return durationLogVal(time.Duration(res))
}

type Log struct {
	Size int
	m    []LogVal
	mux  sync.Mutex
}

// TODO move to separate package
// TODO make only published methods get the lock

func InitLog(s int) *Log {
	return &Log{Size: s, m: make([]LogVal, 0, s)}
}

func (l *Log) Add(val LogVal) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if len(l.m) > l.Size {
		l.m = l.m[1:]
	}

	l.m = append(l.m, val)
}

func (l *Log) Len() int {
	l.mux.Lock()
	defer l.mux.Unlock()
	return len(l.m)
}

func (l *Log) Ends() (LogVal, LogVal, error) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if len(l.m) > 1 {
		return l.m[0], l.m[len(l.m)-1], nil
	}
	return intLogVal(0), intLogVal(0), fmt.Errorf("not enough values")
}

func (l *Log) Latest() (LogVal, error) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if len(l.m) > 0 {
		return l.m[len(l.m)-1], nil
	}
	return intLogVal(0), fmt.Errorf("empty log")
}

func (l *Log) Min() (LogVal, error) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if len(l.m) == 0 {
		return intLogVal(0), fmt.Errorf("empty log")
	}

	least := l.m[0]
	for _, val := range l.m {
		if val.lessthan(least) {
			least = val
		}
	}
	return least, nil
}

func (l *Log) Avg() (LogVal, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	n := len(l.m)
	if n == 0 {
		return intLogVal(0), fmt.Errorf("empty log")
	}

	var sum LogVal
	var dv LogVal
	switch l.m[0].(type) {
	case intLogVal:
		sum = intLogVal(0)
		dv = intLogVal(n)
	case floatLogVal:
		sum = floatLogVal(0)
		dv = floatLogVal(float64(n))
	case durationLogVal:
		sum = durationLogVal(time.Duration(0))
		dv = floatLogVal(time.Duration(n))
	}

	for _, v := range l.m {
		sum = sum.add(v)
	}

	return sum.div(dv), nil
}
