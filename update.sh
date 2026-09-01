#!/usr/bin/env bash

set -e
VERSION=$1
RELEASE=$(date +%Y%m%d)

RPM=~/rpmbuild/RPMS/x86_64/checktests-${VERSION}-${RELEASE}.x86_64.rpm

rpmbuild -bb ~/rpmbuild/SPECS/checktests.spec \
  --define "version ${VERSION}" \
  --define "release ${RELEASE}"

sudo cp "$RPM" /srv/localrepo/
sudo createrepo_c --update /srv/localrepo/

echo "Published checktests-${VERSION} to localrepo"
