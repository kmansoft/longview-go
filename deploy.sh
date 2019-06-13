#!/usr/bin/bash

SERVER="kman.mobi"

echo "*** Building ..."

if ! go build -o longview-go.out *.go
then
	echo "*** Error"
	exit 1
fi

echo "*** Copying ..."

rsync -acv *.service "root@${SERVER}:/etc/systemd/system/"
rsync -acv *.out "root@${SERVER}:"

echo "*** Restarting ..."

systemctl --host "root@${SERVER}" daemon-reload
systemctl --host "root@${SERVER}" restart longview-go
