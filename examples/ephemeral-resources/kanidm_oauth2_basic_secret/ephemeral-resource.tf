resource "kanidm_oauth2_basic" "grafana" {
  name        = "grafana"
  displayname = "Grafana"
  origin      = "https://grafana.example.com"
}

ephemeral "kanidm_oauth2_basic_secret" "grafana" {
  client = kanidm_oauth2_basic.grafana.id
}

resource "vault_kv_secret_v2" "grafana_oidc" {
  mount        = "secret"
  name         = "grafana/oidc"
  disable_read = true
  data_json_wo = jsonencode({
    client_id     = ephemeral.kanidm_oauth2_basic_secret.grafana.client_id
    client_secret = ephemeral.kanidm_oauth2_basic_secret.grafana.client_secret
  })
  data_json_wo_version = 1 # bump after rotating the secret in Kanidm
}
