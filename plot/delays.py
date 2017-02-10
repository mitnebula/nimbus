#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from switches import readSwitches, vlines

plt.cla()
plt.clf()

def readDelays(tr):
    for r in (t for t in tr if 'rtt' in t and 'yt' in t and 'oldYt' in t):
        yield (r['elapsed'], r['rtt'], r['yt'], r['oldYt'])

def makeDelayPlot(name, tr, figInd):
    sw = readSwitches(tr)
    t, xt, yt, ytShift = zip(*readDelays(tr))

    fig1 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('RTT (s)')
    vlines(plt, sw)
    plt.title(name)
    plt.plot(t, xt, label='RTT')

    figInd += 1

    fig2 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('yt (s)')
    vlines(plt, sw)
    plt.title(name)
    plt.plot(t, ytShift, label='y(t) shift')

if __name__ == '__main__':
    tr = list(readTrace(sys.argv[1]))
    makeDelayPlot(sys.argv[1], tr, 1)
    plt.show()
