#!/bin/bash

ARCH_LIST="386 amd64"
if [[ $# -ge 1 ]]
then
    ARCH_LIST="$1"
fi

PACKDIR="./package"
TEMPDIR="${PACKDIR}/temp-debian"
EXENAME="longview-go.out"

VERSION=`cat ${PACKDIR}/VAR_VERSION | perl -ne 'chomp and print'`
BUILD=`cat ${PACKDIR}/VAR_BUILD | perl -ne 'chomp and print'`

[[ -d "${TEMPDIR}" ]] && rm -rf "${TEMPDIR}"

mkdir -p "${TEMPDIR}" && echo "*** Created : ${TEMPDIR}"

mkdir -p "${TEMPDIR}/usr/sbin/"
mkdir -p "${TEMPDIR}/etc/"
mkdir -p "${TEMPDIR}/etc/linode"
mkdir -p "${TEMPDIR}/etc/linode/longview.d"
mkdir -p "${TEMPDIR}/lib/systemd/system/"

# For now this will do
for ARCH in $ARCH_LIST
do
	if [[ "$ARCH" == "386" ]]; then
	    DEB_ARCH=i386
	elif [[ "$ARCH" == "amd64" ]]; then
	    DEB_ARCH=amd64
	else
	    echo "*** Unknown arch $ARCH"
	    exit 1
	fi

	echo "*** Version : ${VERSION}-${BUILD}"
	echo "*** Arch    : ${DEB_ARCH}"

	OUT_DEB="longview-go_${VERSION}-${BUILD}_$DEB_ARCH.deb"

	# Build the binary

	if ! GOARCH="${ARCH}" go build -o "${TEMPDIR}/usr/sbin/${EXENAME}" *.go
	then
		echo "*** Build error"
		exit 1
	fi

	ls -lh "${TEMPDIR}/usr/sbin/${EXENAME}"
	file "${TEMPDIR}/usr/sbin/${EXENAME}"

	# Copy various supporting files

	cp "${PACKDIR}/longview-go.service" "${TEMPDIR}/lib/systemd/system/"

	# Generate debian-binary

	echo "2.0" > "${TEMPDIR}/debian-binary"

	# Generate control

	echo "Version: $VERSION-$BUILD" > "${TEMPDIR}/control"
	echo "Installed-Size:" `du -sb "${TEMPDIR}" | awk '{print int($1/1024)}'` >> "${TEMPDIR}/control"
	echo "Architecture: $DEB_ARCH" >> "${TEMPDIR}/control"
	cat "${PACKDIR}/deb_control" >> "${TEMPDIR}/control"

	# Copy postinst and postrm

	cp "${PACKDIR}/deb_postinst" "${TEMPDIR}/postinst"
	cp "${PACKDIR}/deb_postrm" "${TEMPDIR}/postrm"

	(
	    # Generate md5 sums

	    cd "${TEMPDIR}"

	    find ./usr ./lib ./etc -type f | while read i ; do
	        md5sum "$i" | sed 's/\.\///g' >> md5sums
	    done

	    # Archive control

	    chmod 644 control md5sums
	    chmod 755 postrm postinst
	    fakeroot -- tar -cz -f ./control.tar.gz ./control ./md5sums ./postinst ./postrm

	    # Archive data

	    fakeroot -- tar -cz -f ./data.tar.gz ./etc ./lib ./usr

	    # Make final archive

	    fakeroot -- ar -cr "${OUT_DEB}" debian-binary control.tar.gz data.tar.gz
	)

	ls -lh "${TEMPDIR}/${OUT_DEB}"

	ar -t -f "${TEMPDIR}/${OUT_DEB}"
done
