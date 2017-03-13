#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from switches import readSwitches, vlines

plt.cla()
plt.clf()

def readDelays(tr):
    for r in (t for t in tr if 'rtt' in t):
        yield (r['elapsed'], r['rtt'])

def makeDelayPlot(name, tr, figInd):
    sw = readSwitches(tr)
    t, xt = zip(*readDelays(tr))

    fig1 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('RTT (s)')
    vlines(plt, sw)
    plt.title(name)
    plt.plot(t, xt, label='RTT')

    figInd += 1

    '''fig2 = plt.figure(figInd)
    plt.xlabel('Time (s)')
    plt.ylabel('yt (s)')
    #vlines(plt, sw)
    plt.title(name)
    plt.plot(t, ytShift, label='y(t) shift')

    figInd += 1'''

    return figInd

if __name__ == '__main__':
    tr = list(readTrace(sys.argv[1]))
    makeDelayPlot(sys.argv[1], tr, 1)
    plt.show()
