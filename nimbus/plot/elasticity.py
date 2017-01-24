#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from elasticity_all import read, vlines, elast, switches
from read import parseTime

plt.cla()
plt.clf()

def readElast():
    with open(sys.argv[1], 'r') as f:
        for line in f:
            sp = line.split()
            if sp[0] == 'ELASTICITY:':
                if len(sp) != 5:
                    continue
                _, t, sec5, sec2, short = sp
                yield {
                    't': parseTime(t),
                    '5sec': float(sec5),
                    '2sec': float(sec2),
                    'short': float(short),
                }

def sec5(ls):
    for l in ls:
        if 't' in l and '5sec' in l:
            yield l['t'], l['5sec']
def sec2(ls):
    for l in ls:
        if 't' in l and '2sec' in l:
            yield l['t'], l['2sec']
def secShort(ls):
    for l in ls:
        if 't' in l and 'short' in l:
            yield l['t'], l['short']

if __name__ == '__main__':
    ls = list(read())
    els = list(readElast())

    nxa, s5 = zip(*sec5(els))
    _, s2 = zip(*sec2(els))
    _, mr10 = zip(*secShort(els))
    sw = list(switches(ls))

    fig4 = plt.figure(1)
    plt.xlabel('Time (s)')
    plt.ylabel('elastic detector')
    vlines(plt, sw)
    print sys.argv
    plt.title(sys.argv[1])
    plt.plot(nxa, s5, label='5s')
    plt.plot(nxa, s2, label='2s')
    plt.plot(nxa, mr10, label='10 min_rtt')

    plt.legend(loc='lower left')
    plt.show()

