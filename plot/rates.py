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

if __name__ == '__main__':
    tr = readTrace(sys.argv[1])
    t, rin, rout, zt = zip(*readMeasure(tr))
    sw = readSwitches(tr)

    fig1 = plt.figure(1)
    plt.xlabel('Time (s)')
    plt.ylabel('Cross Traffic (Mbps)')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(t, zt, label='Cross Traffic')

    fig2 = plt.figure(2)
    plt.xlabel('Time (s)')
    plt.ylabel('Rate (Mbps)')
    vlines(plt, sw)
    plt.title(sys.argv[1])
    plt.plot(t, rin, label='rin')
    plt.plot(t, rout, label='rout')
    plt.legend()

    plt.show()

