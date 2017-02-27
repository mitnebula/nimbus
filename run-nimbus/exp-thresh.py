#!/usr/bin/python

import subprocess
from exp import runExp

name = 't10'
bw = 24
buf = [2]

pulseSizes = [0.5]#, 0.5, 0.75]
rtts = [100]
initRates = [12]#, 12, 20]
measures = [1]
crossTraffic = ['tcp', 'poisson', 'cbr']

def run(r, p, c, i, m, b):
    runExp(name, bw, r, b, p, c, i, m)

if __name__ == '__main__':
    for r in rtts:
        for p in pulseSizes:
            for c in crossTraffic:
                for i in initRates:
                    for m in measures:
                        for b in buf:
                            run(r, p, c, i, m, b)


