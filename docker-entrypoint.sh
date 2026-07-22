#!/bin/sh
set -e
# Fix ownership of the data directory in case Docker created it as root
# (happens on first deploy when ./data doesn't exist on the host).
chown openlogs:openlogs /data
exec su-exec openlogs /usr/local/bin/openlogs "$@"
