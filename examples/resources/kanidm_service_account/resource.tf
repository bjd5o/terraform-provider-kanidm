# Example: Service account whose tokens are minted by the OpenBao Kanidm secrets engine.
# entry_managed_by is the only entry Kanidm allows to generate API tokens for this account.
resource "kanidm_service_account" "mailer" {
  name             = "mailer"
  displayname      = "Mailer"
  entry_managed_by = "openbao-token-issuers"
}

# Example: Service account for CI
resource "kanidm_service_account" "argocd" {
  name             = "argocd"
  displayname      = "ArgoCD Service Account"
  entry_managed_by = "openbao-token-issuers"
}

# Example: Service account for monitoring
resource "kanidm_service_account" "prometheus" {
  name             = "prometheus"
  displayname      = "Prometheus Monitoring"
  entry_managed_by = "openbao-token-issuers"
}

# Example: Imported existing service account
# Import command: terraform import kanidm_service_account.existing existing_account_id
resource "kanidm_service_account" "existing" {
  name             = "existing-service"
  displayname      = "Existing Service Account"
  entry_managed_by = "openbao-token-issuers"
}
