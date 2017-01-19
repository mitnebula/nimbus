#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import parseTime, parseSwitchOutput

plt.cla()
plt.clf()


def read():
    with open(sys.argv[1], 'r') as f:
        for line in f:
            sp = line.split()
            if '->' in sp:
                yield parseSwitchOutput(sp)
            elif len(sp) == 6:
                if sp[1] != ':':
                    continue
                t = parseTime(sp[0])
                zt = float(sp[2])
                rtt = parseTime(sp[3])
                rin = float(sp[4])
                rout = float(sp[5])
                yield {
                    'time': t,
                    'zt': zt,
                    'rtt': rtt,
                    'rin': rin,
                    'rout': rout,
                }
            else:
                print sp

def zt(ls):
    for l in ls:
        if 'time' in l and 'zt' in l:
            yield l['time'], l['zt']/1e6

def rtt(ls):
    for l in ls:
        if 'time' in l and 'rtt' in l:
            yield l['time'], l['rtt']

def rin(ls):
    for l in ls:
        if 'time' in l and 'rin' in l:
            yield l['time'], l['rin']/1e6

def rout(ls):
    for l in ls:
        if 'time' in l and 'rout' in l:
            yield l['time'], l['rout']/1e6

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

if __name__ == '__main__':
    ls = list(read())

    nxa, zt = zip(*zt(ls))
    _, rtt = zip(*rtt(ls))
    _, rin = zip(*rin(ls))
    _, rout = zip(*rout(ls))
    sw = list(switches(ls))

    fig1 = plt.figure(1)
    plt.xlabel('Time (s)')
    plt.ylabel('zt (Mbps)')
    vlines(plt, sw)
    plt.plot(nxa, zt, label='nimbus')

    fig2 = plt.figure(2)
    plt.xlabel('Time (s)')
    plt.ylabel('rtt (s)')
    vlines(plt, sw)
    plt.plot(nxa, rtt, label='nimbus')

    fig3 = plt.figure(3)
    plt.xlabel('Time (s)')
    plt.ylabel('rin (Mbps)')
    vlines(plt, sw)
    plt.plot(nxa, rin, label='rin')
    plt.plot(nxa, rout, label='rout')
    plt.legend()

    plt.show()

