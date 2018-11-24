#!/bin/bash

cmd=`which go`
if [ -z $cmd ]; then
    cmd=$1
    if [ -z $cmd ] || [ ! -f $cmd ]; then
        echo "go bin exec not found"
        exit 2
    fi
fi

$cmd build
mkdir -p monitor_logs
\cp control monitor_logs/
\cp falcon-log monitor_logs/
\cp -r conf monitor_logs/

