#!/bin/bash

# For now this will do
ARCH=x86_64

PACKDIR="./package"
TEMPDIR="${PACKDIR}/temp-debian"
EXENAME="longview-go.out"

VERSION=`cat ${PACKDIR}/VAR_VERSION | perl -ne 'chomp and print'`
BUILD=`cat ${PACKDIR}/VAR_BUILD | perl -ne 'chomp and print'`

if [[ "$ARCH" == "i686" ]]; then
    DEB_ARCH=i386
elif [[ "$ARCH" == "x86_64" ]]; then
    DEB_ARCH=amd64
else
    echo "*** Unknown arch $ARCH"
    exit 1
fi

OUT_DEB="longview-go_${VERSION}-${BUILD}_$DEB_ARCH.deb"

SERVER="kman.mobi"
if [[ $# -ge 1 ]]
then
    SERVER="$1"
fi

echo "*** Building Debian package ..."

if ! ./package/package_debian.sh amd64
then
	echo "*** Error"
	exit 1
fi

if [[ "$ARCH" == "i686" ]]; then
    DEB_ARCH=i386
elif [[ "$ARCH" == "x86_64" ]]; then
    DEB_ARCH=amd64
else
    echo "*** Unknown arch $ARCH"
    exit 1
fi

echo "*** Version : ${VERSION}-${BUILD}"
echo "*** Arch    : ${DEB_ARCH}"

echo "*** Copying to ${SERVER} ..."

if ! rsync -acv "${TEMPDIR}/${OUT_DEB}" "root@${SERVER}:"
then
	echo "*** Error"
	exit 1
fi

echo "*** Installing ..."

ssh "root@${SERVER}" "apt-get install --reinstall ./${OUT_DEB}"

echo "*** Restarting ..."

systemctl --host "root@${SERVER}" daemon-reload
systemctl --host "root@${SERVER}" restart longview-go

