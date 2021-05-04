#!/bin/bash

[ -d build ] && rm -rf build

[ -d libs/go-sqlite3 ] && rm -rf libs/go-sqlite3

[ -f lnxmon.db ] && rm lnxmon.db
[ -f lnxmon.db-journal ] && rm lnxmon.db-journal

[ -f lnxmoncli ] && rm lnxmoncli
[ -f lnxmoncli.log ] && rm lnxmoncli.log
[ -f lnxmonsrv ] && rm lnxmonsrv
[ -f lnxmonsrv.log ] && rm lnxmonsrv.log
