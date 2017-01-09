package main

import (
	"fmt"
	"testing"
	"time"
)

// the log can store items and retrieve them
func TestAddOne(t *testing.T) {
	dummyPacket := Packet{}
	lg := InitTimedLog(time.Duration(10) * time.Millisecond)

	tm := time.Now()
	lg.Add(tm, dummyPacket)

	p, ti, err := lg.Before(time.Now())
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if p != dummyPacket {
		t.Error("stored data incorrect")
	}

	if !ti.Equal(tm) {
		t.Error("stored timestamp incorrect")
	}
}

// the legnth of the log doesn't exceed its time allotment
func TestTimeLimit(t *testing.T) {
	lg := InitTimedLog(time.Duration(5) * time.Millisecond)

	tm := time.Now()
	stopAdd := tm.Add(time.Duration(50) * time.Millisecond)
	iter := uint32(0)
	for time.Now().Before(stopAdd) {
		iterPkt := Packet{SeqNo: iter}
		lg.Add(time.Now(), iterPkt)
		iter++
	}

	span := lg.times[len(lg.times)-1].Sub(lg.times[0])
	if span > lg.length {
		t.Error("length incorrect:", lg.Len(), span)
	}

	if lg.Len() < 200 {
		t.Error("length too short:", lg.Len(), span)
	}
}

// the log always has at least 100 items
func TestCountLimit(t *testing.T) {
	lg := InitTimedLog(time.Duration(1) * time.Millisecond)

	tm := time.Now()
	stopAdd := tm.Add(time.Duration(300) * time.Millisecond)
	iter := uint32(0)
	for time.Now().Before(stopAdd) {
		iterPkt := Packet{SeqNo: iter}
		lg.Add(time.Now(), iterPkt)
		<-time.After(time.Duration(2) * time.Millisecond)
		iter++
	}

	if lg.Len() < 100 {
		span := lg.times[len(lg.times)-1].Sub(lg.times[0])
		t.Error("length incorrect:", lg.Len(), span)
	}
}
