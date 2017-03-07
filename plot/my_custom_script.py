import sys
import pylab as pl
import numpy as np
import math
import matplotlib.pyplot as plt
import scipy.fftpack
import subprocess
import time as tym
from scipy import signal
def average(x):
    assert len(x) > 0
    return float(sum(x)) / len(x)

def pearson_def(x, y):
    if len(x)!=len(y):
        #print len(x), len(y)
        return 0.0
    assert len(x) == len(y)
    n = len(x)
    assert n > 0
    avg_x = average(x)
    avg_y = average(y)
    diffprod = 0
    xdiff2 = 0
    ydiff2 = 0
    for idx in range(n):
        xdiff = x[idx] - avg_x
        ydiff = y[idx] - avg_y
        diffprod += xdiff * ydiff
        xdiff2 += xdiff * xdiff
        ydiff2 += ydiff * ydiff

    return diffprod / math.sqrt(xdiff2 * ydiff2)

def main():
	#bashCommand = "python read.py "+sys.argv[1]+" elapsed rtt yt zt > temp.tr"
	#print bashCommand
	#process = subprocess.Popen(bashCommand.split(), shell=True)
	#output, error = process.communicate()
	#subprocess.Popen(command)
	X = pl.loadtxt('temp.tr')
	time = X[:,0:1]
	Z = X[:,2:3]
	rtt = X[:,1:2]
	rout =X[:,3:4]
	start=0.0
	Z2 = []
	time2 = []
	rtt2 = []
	rout2 = []
	i=0
	T=0.01
	while i<len(time):
		while i<len(time) and time[i]<start:
			i+=1
			#print time[i]
		if i>= len(time):
			break
		Z2.append(Z[i])
		rtt2.append(rtt[i])
		time2.append(start)
		rout2.append(rout[i])
		start+=T
		#print start, yt[i]

	x = []
	y = []
	for i in range(len(Z2)):
		#if i<2000 or i>8000:
			#continue
		#x.append(i/200.0)
		#y.append(pearson_def(yt2[i:i+1000],Z2[i+int(rtt2[i]*200):i+1000+int(rtt2[i]*200)]))
		if i==int(float(sys.argv[1])/T):
			N=int(float(sys.argv[2])/T)
			rout3=np.linspace(0.0, N*T, N)
			Z3=np.linspace(0.0, N*T, N)
			for j in range(N):
				Z3[j]=Z2[i+j+int(rtt2[i+j]/T)]
				rout3[j]=rout2[i+j]
			x_prime = np.linspace(i*T, i*T+N*T, N)
			plt.figure()
			plt.title('Snapshot of Z used for FFT')
			plt.xlabel('Time (s)')
			plt.ylabel('Z(t)')
			plt.plot(x_prime, Z2[i:i+N])
			xf = np.linspace(0.0, 1.0/(2.0*T), N/2)
			y_prime = signal.detrend(Z3)
			yf = scipy.fftpack.fft(y_prime * np.hanning(y_prime.size))
			Z_FFT_array = np.abs(yf[:N//2])
			plt.figure()
			plt.plot(xf, 2.0/N * np.abs(yf[:N//2]), label='Z')
			y_prime = signal.detrend(rout3)
			yf = scipy.fftpack.fft(y_prime * np.hanning(y_prime.size))
			Rout_FFT_array = np.abs(yf[:N//2])
			plt.plot(xf,  2.0/N * np.abs(yf[:N//2]), label='Rout')
			plt.xlabel('Frequency (hz)')			
			plt.title('FFT for rout and Z')
			plt.legend(bbox_to_anchor=(1.05, 1), loc=0, borderaxespad=0.)
			max_ind = 0			
			for j in range(len(xf)):
				if xf[j]<1.0:
					max_ind=j
					continue
				if xf[j]>10.0:
					break
				if Rout_FFT_array[j]>Rout_FFT_array[max_ind]:
					max_ind=j
			print "Frequency of Peak", xf[max_ind], "Rout Peak", Rout_FFT_array[max_ind], "Z Peak", Z_FFT_array[max_ind]

	plt.figure()
	plt.xlabel('Time (s)')
	plt.ylabel('Z(t)')
	plt.title('Z(t) vs Time')
	plt.plot(time,Z)

	plt.figure()
	plt.xlabel('Time (s)')
	plt.ylabel('Rtt')
	plt.title('Rtt vs Time')
	plt.plot(time2,rtt2)
	'''plt.figure()
	plt.title('Pearson Correlation Coeff with window size')
	plt.xlabel('Window (sec)')
	plt.ylabel('Pearson Coeff')
	plt.plot(x,y)'''
	plt.show()
	#bashCommand = "rm -rf temp.tr"
	#process = subprocess.Popen(bashCommand.split(), stdout=subprocess.PIPE, shell=True)
	#output, error = process.communicate()
if __name__ == "__main__": main()
