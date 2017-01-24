#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from elasticity_all import read, vlines, elast, switches
from read import parseTime

plt.cla()
plt.clf()

def readElast(fn):
    with open(fn, 'r') as f:
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
            yield l['t'], -l['5sec']
def sec2(ls):
    for l in ls:
        if 't' in l and '2sec' in l:
            yield l['t'], -l['2sec']
def secShort(ls):
    for l in ls:
        if 't' in l and 'short' in l:
            yield l['t'], -l['short']

def plotWorkload(fn, ax, title):
    ls = list(read(fn))
    els = list(readElast(fn))

    nxa, s5 = zip(*sec5(els))
    _, s2 = zip(*sec2(els))
    _, mr10 = zip(*secShort(els))
    sw = list(switches(ls))

    ax.set_xlabel('Time (s)')
    ax.set_ylabel('Elasticity')
    #vlines(plt, sw)
    #ax.set_title(title)
    ax.set_xlim(0, 60)
    ax.set_ylim(-0.5, 1.5)
    ax.grid()
    ax.plot(nxa, s5, 'b-', label='5s')
    ax.plot(nxa, s2, 'g-', label='2s')
    ax.plot(nxa, mr10, 'r-', label='10 min_rtt')

    ax.legend(loc='lower left', ncol=3)

if __name__ == '__main__':
    #f, (cbr, tcp, pois) = plt.subplots(1, 3)
    #plotWorkload('e3-cbr.out', cbr, 'Inelastic')
    #plotWorkload('e3-tcp.out', tcp, 'Elastic')
    #plotWorkload('e3-poisson.out', pois, 'Inelastic Poisson')

    fig = plt.figure(1, figsize=(4, 3))
    plotWorkload(sys.argv[1], fig.gca(), sys.argv[2])


    plt.savefig('{}-elasticity.pdf'.format(sys.argv[2]))
