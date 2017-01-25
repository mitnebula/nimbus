package main

import (
	"testing"
	"time"
)

func TestLen(t *testing.T) {
	l := InitLog(10)
	l.Add(intLogVal(42))

	if s := l.Len(); s != 1 {
		t.Error("wrong len", s, "expected 1")
	}
}

func TestLenLimit(t *testing.T) {
	l := InitLog(10)

	for i := 0; i < 20; i++ {
		l.Add(intLogVal(42))
	}

	if s := l.Len(); s != 10 {
		t.Error("wrong len, got", s, "expected 10")
	}
}

func TestEnds(t *testing.T) {
	l := InitLog(10)

	for i := 50; i < 52; i++ {
		l.Add(intLogVal(i))
	}

	old, new, err := l.Ends()
	if err != nil {
		t.Errorf("got error on Ends(): %s", err)
	}

	oldest := int64(old.(intLogVal))
	newest := int64(new.(intLogVal))
	if oldest != 50 || newest != 51 {
		t.Errorf("Ends() values incorrect, got (%d, %d), expected (50, 52)", oldest, newest)
	}
}

func TestLatest(t *testing.T) {
	l := InitLog(10)

	for i := 50; i < 52; i++ {
		l.Add(intLogVal(i))
	}

	_, new, err := l.Ends()
	if err != nil {
		t.Errorf("got error on Ends(): %s", err)
	}

	newest := int64(new.(intLogVal))
	if newest != 51 {
		t.Errorf("Latest() value incorrect, got %d, expected 52", newest)
	}
}

func TestMin(t *testing.T) {
	l := InitLog(10)

	for i := 0; i < 20; i++ {
		l.Add(intLogVal(i))
	}

	s, err := l.Min()
	if err != nil {
		t.Errorf("got error on Min(): %s", err)
	}

	val := int64(s.(intLogVal))
	if val != 10 {
		t.Errorf("Min() value incorrect, got %d, expected 10", val)
	}
}

func TestAvgInt(t *testing.T) {
	l := InitLog(10)

	for i := 0; i < 20; i++ {
		l.Add(intLogVal(i))
	}

	s, err := l.Avg()
	if err != nil {
		t.Errorf("got error on Avg(): %s", err)
	}

	val := int64(s.(intLogVal))
	if val != 14 {
		t.Errorf("Avg() value incorrect, got %d, expected 14.5", val)
	}
}

func TestAvgDur(t *testing.T) {
	l := InitLog(10)

	for i := 0; i < 20; i++ {
		l.Add(durationLogVal(time.Duration(i)))
	}

	s, err := l.Avg()
	if err != nil {
		t.Errorf("got error on Avg(): %s", err)
	}

	val := time.Duration(s.(durationLogVal))
	if val != time.Duration(14) {
		t.Errorf("Avg() value incorrect, got %d, expected 14.5", val)
	}
}
