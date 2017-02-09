#!/usr/bin/python

import sys
import matplotlib
from matplotlib import pyplot as plt
import numpy as np

from elasticity import readElast, sec5

matplotlib.rc('font', family='sans-serif')
matplotlib.rc('font', serif='Futura Medium')
matplotlib.rc('text', usetex='false')
matplotlib.rcParams.update({'font.size': 14})


plt.cla()
plt.clf()

def tots(ls):
    for l in ls:
        if 't' in l and 'tot' in l:
            yield l['t'], -l['tot']

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

    els = list(readElast('./g2-{}.out'.format(sys.argv[1])))
    bad, noYt = zip(*tots(els))

    els = list(readElast('./g1-{}.out'.format(sys.argv[1])))
    good, yt = zip(*tots(els))

    plt.plot(bad, noYt, label='Using x(t)')
    plt.plot(good, yt, label='Using y(t)')

    plt.legend(loc='upper left')
    plt.savefig('compareYt-{}.pdf'.format(sys.argv[1]))
