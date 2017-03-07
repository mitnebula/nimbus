#!/usr/bin/python

#from read import readTrace

def readSwitches(tr):
    for r in (t for t in tr if 'to' in t and 'elapsed' in t):
        yield (r['elapsed'], r['to'])
	

def vlines(p, sw):
    for t, to in sw:
        if to == 'DELAY':
            p.axvline(t, color='red')
        elif 'XTCP' in to:
            p.axvline(t, color='green')
        else:
            p.axvline(t, color='gray')
