#!/bin/bash

VERSION=${CIRCLE_TAG##v}

# Prepare the RPM spec
sed -i "s:VERSION:${VERSION}:g" circleci-rpmspec.spec

# Build the RPM
rpmbuild -bb circleci-rpmspec.spec

# Copy the RPM(s) back down to the workspace
# for later insertion into packagecloud
cp ~/rpmbuild/*.rpm /srv
