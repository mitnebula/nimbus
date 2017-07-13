#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from switches import readSwitches, vlines

plt.cla()
plt.clf()



def readMeasure(tr):
    for r in (t for t in tr if 'Elasticity' in t):
        if r['elapsed']<30.0 or r['elapsed']>90.0:
            continue
        yield (r['elapsed']-30.0, r['Elasticity'])

def makeRatesPlot(tr, lab):
    t, e = zip(*readMeasure(tr))
    plt.plot(t, e, label=lab)

if __name__ == '__main__':
    plt.xlabel('Time (s)')
    plt.ylabel('Elasticity')
    for i in range(int(sys.argv[1])):
        tr = list(readTrace(sys.argv[2*i+2]))
        makeRatesPlot(tr, sys.argv[2*i+3])
    plt.legend(bbox_to_anchor=(1.05, 1), loc=1, borderaxespad=0.)
    axes = plt.gca()
    axes.set_ylim([-0.5,1.5])
    plt.show()

