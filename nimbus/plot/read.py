#!/usr/bin/python

import sys

def readTime(t):
    if t[-2:] == "ms":
        return float(t[:-3]) * 1e3
    elif t[-1:] == "s":
        return float(t[:-3])
    else:
        assert False

def readLines():
    for line in sys.stdin:
        sp = line.split()
        if len(sp) != 14:
            print sp
            continue
        _, time, _, oldR, _, currR, _, rin, _, zt, _, minrtt, _, rtt = sp
        yield {
            't': int(time),
            'rate': float(currR),
            'cross': float(zt),
            'minrtt': readTime(minrtt),
            'rtt': readTime(rtt),
        }
