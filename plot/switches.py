#!/usr/bin/python

from read import readTrace

def readSwitches(tr):
    for s in (t for t in tr if 'from' in t and 'to' in t):
        yield (s['elapsed'], s['to'])

def vlines(p, sw):
    for t, to in sw:
        if to == 'DELAY':
            p.axvline(t, color='red')
        elif 'XTCP' in to:
            p.axvline(t, color='green')
        else:
            p.axvline(t, color='gray')
