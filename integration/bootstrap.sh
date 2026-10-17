#!/usr/bin/env bash
# Start Kanidm, mint a service-account token for OpenTofu, and write it
# to the shared kanidm-token volume at /run/kanidm/token.
#
# The idm_admin password stays in this process. The token is not printed.
#
#   integration/bootstrap.sh
#   docker compose -f integration/compose.yaml run --rm opentofu tofu version
#
# Re-running mints a new token and overwrites the file. Restart the
# opentofu service afterwards so it reads the new one.
set -euo pipefail
cd "$(dirname "$0")"

PORT=${ITEST_KANIDM_PORT:-18443}
ACCOUNT=tofu
COMPOSE=(docker compose -f compose.yaml)

log() { printf '==> %s\n' "$*" >&2; }

dc() { "${COMPOSE[@]}" "$@"; }

k() { dc exec -T -e KANIDM_PASSWORD tools kanidm "$@" -D idm_admin; }

exists() { # group|service-account name
	k "$1" get "$2" 2>/dev/null | grep -q "^name: $2\$"
}

add_member() { # group
	if k group get "$1" 2>/dev/null | grep -q "^member: ${ACCOUNT}@"; then
		return 0
	fi
	k group add-members "$1" "$ACCOUNT" >/dev/null
}

log "starting kanidm"
dc up -d kanidm tools

log "waiting for kanidm"
ready=0
for _ in $(seq 1 90); do
	if curl -sfk "https://127.0.0.1:${PORT}/status" >/dev/null; then
		ready=1
		break
	fi
	sleep 1
done
if [[ $ready -ne 1 ]]; then
	log "timed out waiting for kanidm on port $PORT"
	exit 1
fi

log "recovering idm_admin"
ADMIN_PASSWORD=$(dc exec -T kanidm kanidmd recover-account idm_admin 2>&1 | sed -E 's/\x1b\[[0-9;]*m//g' | sed -n 's/.*new_password: "\(.*\)".*/\1/p' | tail -n1)
[[ -n $ADMIN_PASSWORD ]] || { log "could not read the new idm_admin password"; exit 1; }
export KANIDM_PASSWORD=$ADMIN_PASSWORD

cleanup() {
	if [[ -n ${KANIDM_PASSWORD:-} ]]; then
		k logout >/dev/null 2>&1 || true
	fi
	unset KANIDM_PASSWORD ADMIN_PASSWORD TOKEN
}
trap cleanup EXIT

k login >/dev/null

# Kanidm 1.11 sets credential_type_minimum=mfa on idm_all_persons, so a
# password alone cannot be committed. The acceptance tests log in with a
# password, which requires the weaker minimum.
log "allowing password credentials for persons"
k group account-policy credential-type-minimum idm_all_persons any >/dev/null

if ! exists service-account "$ACCOUNT"; then
	log "creating service account $ACCOUNT"
	k service-account create "$ACCOUNT" "OpenTofu" idm_admins >/dev/null
fi

log "minting a token for $ACCOUNT"
# idm_admin can mint because entry_managed_by is idm_admins. Do this
# before the account joins privileged groups.
TOKEN=$(k service-account api-token generate "$ACCOUNT" provider --readwrite 2>/dev/null | tail -n1 | tr -d '[:space:]')
[[ $TOKEN =~ ^[A-Za-z0-9._-]{40,}$ ]] || { log "could not mint the service account token"; exit 1; }

log "granting $ACCOUNT admin rights"
add_member idm_oauth2_admins
add_member idm_admins

log "writing the token to the shared volume"
printf '%s' "$TOKEN" | dc exec -T tools sh -c 'cat > /run/kanidm/token && chmod 644 /run/kanidm/token'
unset TOKEN

log "starting opentofu"
dc up -d opentofu
log "token is at /run/kanidm/token in the opentofu service"
