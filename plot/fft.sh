#!/bin/bash

python read.py $1 elapsed rtt yt zt rout > temp.tr
python my_custom_script.py
rm -rf temp.tr
