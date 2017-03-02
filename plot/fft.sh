#!/bin/bash

python read.py $1 elapsed rtt zt rout > temp.tr
python my_custom_script.py $2 $3
rm -rf temp.tr
