#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import parseTime, parseSwitchOutput

plt.cla()
plt.clf()


def read(fn):
    with open(fn, 'r') as f:
        for line in f:
            sp = line.split()
            if '->' in sp:
                yield parseSwitchOutput(sp)
            elif len(sp) == 9:
                if sp[1] != ':':
                    continue
                t = parseTime(sp[0])
                zt = float(sp[2])
                rtt = parseTime(sp[3])
                rin = float(sp[4])
                rout = float(sp[5])
                elast = float(sp[6])
                setRate = float(sp[7])
                yt = parseTime(sp[8])
                yield {
                    'time': t,
                    'zt': zt,
                    'rtt': rtt,
                    'rin': rin,
                    'rout': rout,
                    'elast': elast,
                    'fr': setRate,
                    'yt': yt,
                }

def zt(ls):
    for l in ls:
        if 'time' in l and 'zt' in l:
            yield l['time'], l['zt']/1e6

def rtt(ls):
    for l in ls:
        if 'time' in l and 'rtt' in l:
            yield l['time'], l['rtt']

def elast(ls):
    for l in ls:
        if 'time' in l and 'elast' in l:
            yield l['time'], l['elast']

def rin(ls):
    for l in ls:
        if 'time' in l and 'rin' in l:
            yield l['time'], l['rin']/1e6

def rout(ls):
    for l in ls:
        if 'time' in l and 'rout' in l:
            yield l['time'], l['rout']/1e6

def setRate(ls):
    for l in ls:
        if 'time' in l and 'fr' in l:
            yield l['time'], l['fr']/1e6

def switches(ls):
    for l in ls:
        if 'from' in l and 'to' in l and 'time' in l:
            yield l['time'], l['to']

def vlines(p, sw):
    for t, to in sw:
        if to == 'DELAY':
            p.axvline(t, color='red')
        elif 'XTCP' in to:
            p.axvline(t, color='green')
        else:
            p.axvline(t, color='gray')

def derivative(times, nums):
    x = zip(times, nums)
    curr = x[0]
    for n in x[1:]:
        dn = n[1] - curr[1]
        yield dn
        curr = n

def integral(horizon, nums):
    buf = []
    for n in nums:
        buf.append(n)
        if len(buf) > horizon:
            buf = buf[1:]
        yield sum(buf)

if __name__ == '__main__':
    ls = list(read(sys.argv[1]))

    nxa, zt = zip(*zt(ls))
    _, rtt = zip(*rtt(ls))
    _, rin = zip(*rin(ls))
    _, rout = zip(*rout(ls))
    _, elast = zip(*elast(ls))
    _, fr = zip(*setRate(ls))
    sw = list(switches(ls))

    fig1 = plt.figure(1)
    plt.xlabel('Time (s)')
    plt.ylabel('zt (Mbps)')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(nxa, zt, label='nimbus')

    zs = [z for z in zt if z != 0]

    fig2 = plt.figure(2)
    plt.xlabel('Time (s)')
    plt.ylabel('rtt (s)')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(nxa, rtt, label='nimbus')

    fig3 = plt.figure(3)
    plt.xlabel('Time (s)')
    plt.ylabel('rin (Mbps)')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(nxa, rin, label='rin')
    plt.plot(nxa, rout, label='rout')
    #plt.plot(nxa, fr, label='set')
    plt.legend()

    fig4 = plt.figure(4)
    plt.xlabel('Time (s)')
    plt.ylabel('elastic detector')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(nxa, elast, label='nimbus')

    plt.show()

