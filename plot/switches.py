#!/usr/bin/python

#from read import readTrace

def readSwitches(tr):
    for r in (t for t in tr if 'to' in t and 'elapsed' in t):
        yield (r['elapsed'], r['to'])

def readMode(tr):
    for r in (t for t in tr if 'initMode' in t):
        return r['initMode']

def vlines(p, sw, duration, initMode):
    last_switch = 14.99
    current_mode=initMode
    total_timein_XTCP = 0.0
    for t, to in sw:
        if to == 'DELAY':
            p.axvline(t, color='red')
            total_timein_XTCP += t - last_switch
            last_switch=t
            current_mode="DELAY"
        elif 'XTCP' in to:
            p.axvline(t, color='green')
            last_switch=t
            current_mode="XTCP"
        else:
            p.axvline(t, color='gray')
    if current_mode=="XTCP":
        total_timein_XTCP += duration - last_switch
    return total_timein_XTCP