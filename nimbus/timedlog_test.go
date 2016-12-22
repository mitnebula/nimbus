package main

import (
	"testing"
	"time"
)

func TestTimeLimit(t *testing.T) {
	dur := time.Duration(10) * time.Millisecond
	l := InitTimedLog(dur)

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
	l := InitTimedLog(startDur)

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
