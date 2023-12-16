#!/bin/bash

times=0

while [ $times -lt 5 ]
do
    wrk -c 500 -t 10 http://127.0.0.1:8080
    times=$[$times+1]
    sleep 2
    echo "--------------------------------------"
done
