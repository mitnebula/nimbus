package main

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeLimit(t *testing.T) {
	dur := time.Duration(10) * time.Millisecond
	l := InitTimedLog(dur, 0)

	start := time.Now()
	for i := time.Now(); i.Before(start.Add(dur * 2)); i = time.Now() {
		l.Add(time.Now(), intLogVal(42))
	}

	if s := CurrSpan(l.times); s > dur {
		t.Errorf("oldest logged: %v ago, expected duration: %v", s, dur)
	}
}

func TestChangingTimeLimit(t *testing.T) {
	startDur := time.Duration(10) * time.Millisecond
	realDur := time.Duration(15) * time.Millisecond
	l := InitTimedLog(startDur, 0)

	start := time.Now()
	for i := time.Now(); i.Before(start.Add(startDur * 5)); i = time.Now() {
		if i.After(start.Add(realDur)) && l.length != realDur {
			l.length = realDur
		}

		l.Add(time.Now(), intLogVal(42))
	}

	if s := CurrSpan(l.times); s > realDur {
		t.Errorf("oldest logged: %v ago, expected duration: %v", s, time.Duration(15)*time.Second)
	}

	if s := CurrSpan(l.times); s < time.Duration(14)*time.Millisecond {
		t.Errorf("oldest logged: %v ago, expected duration: %v", s, time.Duration(15)*time.Second)
	}
}

func TestDelayQuery(t *testing.T) {
	dur := time.Duration(10) * time.Millisecond
	sl := time.Duration(3) * time.Millisecond
	l := InitTimedLog(dur, sl)

	start := time.Now()
	for i := time.Now(); i.Before(start.Add(dur * 2)); i = time.Now() {
		l.Add(time.Now(), intLogVal(42))
	}

	_, _, err := l.Latest(time.Duration(14) * time.Millisecond)
	if err == nil {
		t.Errorf("expected error: delay too large on delayed latest query")
		return
	}

	v, i, err := l.Latest(sl)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
		return
	}

	if v.(intLogVal) != 42 {
		t.Errorf("unexpected value")
		return
	}

	bound := start.Add(-1 * time.Duration(3) * time.Millisecond)
	if i.Before(bound) {
		t.Errorf("past delay value: %v %v", i, bound)
		return
	}
}
