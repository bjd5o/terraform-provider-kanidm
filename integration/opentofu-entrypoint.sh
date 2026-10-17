#!/bin/sh
# Wait until bootstrap has written the service-account token, trust
# Kanidm's CA, then run the given command with KANIDM_TOKEN set.
set -eu

token_file=/run/kanidm/token
i=0
while [ ! -s "$token_file" ]; do
	i=$((i + 1))
	if [ "$i" -gt 300 ]; then
		echo "timed out waiting for $token_file" >&2
		exit 1
	fi
	sleep 1
done

export KANIDM_TOKEN=$(cat "$token_file")
export KANIDM_URL="${KANIDM_URL:-https://idm.test:8443}"
export KANIDM_CA_PATH="${KANIDM_CA_PATH:-/data/ca.pem}"

if [ -f "$KANIDM_CA_PATH" ] && [ -d /usr/local/share/ca-certificates ] && command -v update-ca-certificates >/dev/null 2>&1; then
	cp "$KANIDM_CA_PATH" /usr/local/share/ca-certificates/kanidm.crt
	update-ca-certificates >/dev/null 2>&1 || true
fi

exec "$@"
