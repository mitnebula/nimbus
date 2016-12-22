package main

import (
	"fmt"
	"sync"
)

type LogVal interface {
	lessthan(o LogVal) bool
}

type intLogVal int64

func (i intLogVal) lessthan(o LogVal) bool {
	return int64(i) < int64(o.(intLogVal))
}

type Log struct {
	Size int
	m    []LogVal
	mux  sync.Mutex
}

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
