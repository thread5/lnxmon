#!/bin/bash


function start {
  nohup ./lnxmonsrv >>./lnxmonsrv.log 2>&1 &
}


function stop {
  pid=$(ps aux |grep "./lnxmonsrv" |grep -v "grep" |awk '{print $2}')
  if [ ! -z ${pid} ]; then
    kill -9 ${pid}
  fi
}


function status {
  ps axu |grep "./lnxmonsrv" |grep -v "grep"
}


case $1 in
  start)
    start
    status
    ;;
  stop)
    stop
    status
    ;;
  restart)
    stop
    start
    status
    ;;
  status)
    status
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|status}"
    ;;
esac
