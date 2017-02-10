#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from switches import readSwitches, vlines

plt.cla()
plt.clf()

def readMeasure(tr):
    for r in (t for t in tr if 'rin' in t and 'rout' in t and 'zt' in t):
        yield (r['elapsed'], r['rin']/1e6, r['rout']/1e6, r['zt']/1e6)

def makeRatesPlot(name, tr, figInd):
    t, rin, rout, zt = zip(*readMeasure(tr))
    sw = readSwitches(tr)

    fig1 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('Cross Traffic (Mbps)')
    vlines(plt, sw)
    plt.title(name)
    plt.plot(t, zt, label='Cross Traffic')

    figInd += 1

    fig2 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('Rate (Mbps)')
    vlines(plt, sw)
    plt.title(name)
    plt.plot(t, rin, label='rin')
    plt.plot(t, rout, label='rout')
    plt.legend()

    return figInd

if __name__ == '__main__':
    tr = readTrace(sys.argv[1])
    makeRatesPlot(sys.argv[1], tr, 1)
    plt.show()
