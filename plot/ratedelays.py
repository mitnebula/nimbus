#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readTrace
from rates import makeRatesPlot
from delays import makeDelayPlot

if __name__ == '__main__':
    tr = list(readTrace(sys.argv[1]))
    ind = makeRatesPlot(sys.argv[1], tr, 1)
    f = plt.figure()
    makeDelayPlot(sys.argv[1], tr, ind)
    plt.show()

