#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from elasticity import readElast, sec5

import matplotlib
matplotlib.rc('font', family='sans-serif')
matplotlib.rc('font', serif='Futura Medium')
matplotlib.rc('text', usetex='false')
matplotlib.rcParams.update({'font.size': 14})


plt.cla()
plt.clf()

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

    ax.legend(loc='lower left', ncol=3)

if __name__ == '__main__':
    fig = plt.figure(1, figsize=(4, 3))

    plt.xlabel('Time (s)')
    plt.ylabel('Elasticity')

    plt.xlim(0, 60)
    plt.ylim(-0.5, 1.5)
    plt.gca().grid()

    print sys.argv[1]

    els = list(readElast('./e5-{}.out'.format(sys.argv[1])))
    npAx, noPulse = zip(*sec5(els))

    els = list(readElast('./e4-{}.out'.format(sys.argv[1])))
    mpAx, midPulse = zip(*sec5(els))

    els = list(readElast('./e3-{}.out'.format(sys.argv[1])))
    lpAx, bigPulse = zip(*sec5(els))

    plt.plot(lpAx, bigPulse, label='0.75 Pulse Size')
    plt.plot(mpAx, midPulse, label='0.5 Pulse Size')
    plt.plot(npAx, noPulse, label='No Pulses')

    plt.legend(loc='upper left')
    plt.savefig('comparePulses-{}.pdf'.format(sys.argv[1]))
