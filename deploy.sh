#!/usr/bin/bash

SERVER="kman.mobi"
if [ $# -gt 0 ]
then
	SERVER="$1"
fi


echo "*** Building ..."

if ! go build -o longview-go.out *.go
then
	echo "*** Error"
	exit 1
fi

echo "*** Copying to ${SERVER}..."

if ! rsync -acv *.service "root@${SERVER}:/etc/systemd/system/" || ! rsync -acv *.out "root@${SERVER}:"
then
	echo "*** Error"
	exit 1
fi

echo "*** Restarting ..."

systemctl --host "root@${SERVER}" daemon-reload
systemctl --host "root@${SERVER}" restart longview-go
