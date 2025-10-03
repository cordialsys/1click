#!/bin/bash
set -ex

# This should be run by a systemd unit before the docker systemd unit.
# https://en.wikipedia.org/wiki/Uname
if [ "${HOST_OS}" = "Darwin" ] ; then

  echo "Configuring VFS storage driver for MacOS host"
  mkdir -p /etc/docker
  # Use vfs for testing locally on macos (docker-in-docker).
  # Otherwise, it doesn't work with other drivers.  VFS is slow and
  # we shouldn't use it in prod.
  echo '{ "storage-driver": "vfs" }' > /etc/docker/daemon.json

fi

