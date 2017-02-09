#!/usr/bin/python

import sys
import math
import matplotlib
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from switches import readSwitches, vlines

matplotlib.rc('font', family='sans-serif')
matplotlib.rc('font', serif='Futura Medium')
matplotlib.rc('text', usetex='false')
matplotlib.rcParams.update({'font.size': 14})

plt.cla()
plt.clf()

def readElast(tr):
    for e in (t for t in tr if any('elast' in k for k in t.keys())):
        if 'elast_5sec' in e and 'msg' in e and e['msg'] == 'ELASTICITY':
            yield (e['elapsed'], -e['elast_5sec'])

if __name__ == '__main__':
    tr = list(readTrace(sys.argv[1]))
    t, el5 = zip(*readElast(tr))
    sw = list(readSwitches(tr))
    print 'sw', len(sw)

    fig = plt.figure(1)
    ax = fig.gca()
    ax.set_xlabel('Time (s)')
    ax.set_ylabel('Elasticity')
    ax.set_xlim(0, 60)
    ax.set_ylim(-0.5, 1.5)
    ax.grid()
    vlines(ax, sw)
    ax.plot(t, el5, 'b-')

    plt.show()
