### Migration guide

This repo was forked from https://github.com/seanlatimer/terraform-provider-kanidm with minor changes to improve the security posture.
Specifically, everything which previously wrote a credential to state or plan has been removed:

- `kanidm_service_account` no longer generates or stores an API token. `generate_api_token` and `api_token` are removed. Because they are read-once, there is no way to generate them via Terraform without also writing them to state, so they should be created out-of-band, either manually or with a secrets manager like OpenBao with a Kanidm secrets engine plugin.
- `kanidm_person` no longer generates a credential reset token. `generate_initial_credential_reset_token`, `initial_credential_reset_token`, and `initial_credential_reset_token_ttl` are removed. Like the service account API token, because they are read-once, there is no way to generate them via Terraform without also writing them to state.
- Provider `person_defaults.generate_initial_credential_reset_token` and `person_defaults.initial_credential_reset_token_ttl` are removed. A configuration that still sets either key fails validation.
- `password` is replaced by write-only `password_wo` and `password_wo_version`. The password is never stored. This requires OpenTofu or Terraform 1.11 or later. Increase `password_wo_version` to change the password; changing `password_wo` alone does not produce a plan diff.
- `kanidm_oauth2_basic` no longer reads or stores `client_secret`. Read the secret during a run with the `kanidm_oauth2_basic_secret` ephemeral resource. The provider identity needs membership in `idm_oauth2_admins` only when that ephemeral resource is used.

If you are migrating from the original provider, note that the first refresh migrates state: each changed resource bumps its schema version and an upgrader drops the removed fields. Upgraders cannot reach older state versions the backend kept, so rotate every secret the previous provider wrote.

### Rotation checklist

Do this after the first apply with this release, so a newly rotated secret is never written by the old provider.

- [ ] Revoke every service account API token labelled `terraform-managed`. They have no expiry.
- [ ] Regenerate the secret of every `kanidm_oauth2_basic` client (`POST /v1/oauth2/<name>/_basic_secret`), then publish the new secret with `kanidm_oauth2_basic_secret`.
- [ ] Reset the password of every person whose password was set through OpenTofu or Terraform.
- [ ] Reset tokens from `initial_credential_reset_token` expire after their TTL (default 1 hour) and need no action.
- [ ] Delete old state versions from the backend if it allows it (S3 versions, local `terraform.tfstate.backup`).
