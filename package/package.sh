#!/bin/bash

ARCH_LIST="386 amd64"
if [[ $# -ge 1 ]]
then
    ARCH_LIST="$1"
fi

PACKDIR="./package"
TEMPDIR_DEB="${PACKDIR}/temp-deb"
TEMPDIR_RPM="${PACKDIR}/temp-rpm"
OUTDIR="${PACKDIR}/out"
EXENAME="longview-go.out"

VERSION=`cat ${PACKDIR}/VAR_VERSION | perl -ne 'chomp and print'`
BUILD=`cat ${PACKDIR}/VAR_BUILD | perl -ne 'chomp and print'`

[[ -d "${TEMPDIR_DEB}" ]] && rm -rf "${TEMPDIR_DEB}"
[[ -d "${TEMPDIR_RPM}" ]] && rm -rf "${TEMPDIR_RPM}"

mkdir -p "${TEMPDIR_DEB}" && echo "*** Created : ${TEMPDIR_DEB}"
mkdir -p "${TEMPDIR_RPM}" && echo "*** Created : ${TEMPDIR_RPM}"
mkdir -p "${OUTDIR}" && echo "*** Created : ${OUTDIR}"

mkdir -p "${TEMPDIR_DEB}/usr/sbin/"
mkdir -p "${TEMPDIR_DEB}/etc/"
mkdir -p "${TEMPDIR_DEB}/etc/linode"
mkdir -p "${TEMPDIR_DEB}/etc/linode/longview.d"
mkdir -p "${TEMPDIR_DEB}/lib/systemd/system/"

# For now this will do
for ARCH in $ARCH_LIST
do
	if [[ "$ARCH" == "386" ]]; then
	    DEB_ARCH=i386
      RPM_ARCH=i386
	elif [[ "$ARCH" == "amd64" ]]; then
	    DEB_ARCH=amd64
      RPM_ARCH=x86_64
	else
	    echo "*** Unknown arch $ARCH"
	    exit 1
	fi

	echo "*** Version : ${VERSION}-${BUILD}"
	echo "*** Arch    : ${DEB_ARCH}"

	OUT_DEB="longview-go_${VERSION}-${BUILD}_$DEB_ARCH.deb"
  OUT_RPM="longview-go-${VERSION}-${BUILD}.$RPM_ARCH.rpm"

	# Build the binary

	if ! GOARCH="${ARCH}" go build -o "${TEMPDIR_DEB}/usr/sbin/${EXENAME}" *.go
	then
		echo "*** Build error"
		exit 1
	fi

	ls -lh "${TEMPDIR_DEB}/usr/sbin/${EXENAME}"
	file "${TEMPDIR_DEB}/usr/sbin/${EXENAME}"

	# Copy various supporting files

	cp "${PACKDIR}/longview-go.service" "${TEMPDIR_DEB}/lib/systemd/system/"

	# Create config giles

    printf "# Set your API key below like this\n12345678-1234-1234-1223334445556677\n" \
        > "${TEMPDIR_DEB}/etc/linode/longview.key"

    printf "# Set your Apache stats location like this\n#location http://127.0.0.1/server-status?auto\n" \
        > "${TEMPDIR_DEB}/etc/linode/longview.d/Apache.conf"
    printf "# Set your MySQL stats user like this\n#username linode-longview\n#password example_password\n" \
        > "${TEMPDIR_DEB}/etc/linode/longview.d/MySQL.conf"
    printf "# Set your Nginx stats location like this\n#location http://127.0.0.1/nginx_status\n" \
        > "${TEMPDIR_DEB}/etc/linode/longview.d/Nginx.conf"

	# Generate debian-binary

	echo "2.0" > "${TEMPDIR_DEB}/debian-binary"

	# Generate control

	echo "Version: $VERSION-$BUILD" > "${TEMPDIR_DEB}/control"
	echo "Installed-Size:" `du -sb "${TEMPDIR_DEB}" | awk '{print int($1/1024)}'` >> "${TEMPDIR_DEB}/control"
	echo "Architecture: $DEB_ARCH" >> "${TEMPDIR_DEB}/control"
	cat "${PACKDIR}/deb/control" >> "${TEMPDIR_DEB}/control"

	# Copy conffile

	cp "${PACKDIR}/deb/conffile" "${TEMPDIR_DEB}/conffile"

	# Copy postinst and postrm

	cp "${PACKDIR}/deb/postinst" "${TEMPDIR_DEB}/postinst"
	cp "${PACKDIR}/deb/postrm" "${TEMPDIR_DEB}/postrm"

	(
	    # Generate md5 sums

	    cd "${TEMPDIR_DEB}"

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

	    fakeroot -- ar -cr "../out/${OUT_DEB}" debian-binary control.tar.gz data.tar.gz

	    # Sign it

	    if which debsigs 2> /dev/null
	    then
	    	debsigs --sign=origin --default-key=20AE9981FBC18F91 "../out/${OUT_DEB}"
	    fi
	)

	ls -lh "${OUTDIR}/${OUT_DEB}"

	# RPM

            mkdir -p "${TEMPDIR_RPM}//SPECS"
            cat > "${TEMPDIR_RPM}/SPECS/longview-go.spec" <<EOF
Summary: Longview agent, unofficial
Name: longview-go
Version: ${VERSION}
Release: ${BUILD}
License: GPL
URL: https://github.com/kmansoft/longview-go
Group: System
Packager: Kostya Vasilyev <kmansoft@gmail.com>
Requires: libpthread
Requires: libc

%description
Longview is an alternative client for Linode Longview, written in GO, no install dependencies.

%files
%attr(0744, root, root) /usr/sbin/*
%attr(0644, root, root) /lib/systemd/system/*
%attr(0644, root, root) /etc/linode/*

%prep
echo "BUILDROOT = \$RPM_BUILD_ROOT"

mkdir -p \$RPM_BUILD_ROOT/etc/
mkdir -p \$RPM_BUILD_ROOT/usr/sbin/
mkdir -p \$RPM_BUILD_ROOT/lib/systemd/system/

cp -r ${PWD}/package/temp-deb/etc \$RPM_BUILD_ROOT/
cp ${PWD}/package/temp-deb/usr/sbin/${EXENAME} \$RPM_BUILD_ROOT/usr/sbin/
cp ${PWD}/package/longview-go.service \$RPM_BUILD_ROOT/lib/systemd/system/
EOF

            cat "${TEMPDIR_RPM}/SPECS/longview-go.spec"

            rpmbuild -bb --target "${RPM_ARCH}" \
                "${TEMPDIR_RPM}/SPECS/longview-go.spec"

            cp "${HOME}/rpmbuild/RPMS/${RPM_ARCH}/${OUT_RPM}" "${OUTDIR}/${OUT_RPM}"

done # ARCH
