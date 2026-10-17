package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ssoriche/terraform-provider-kanidm/internal/client"
)

var (
	_ ephemeral.EphemeralResource              = (*oauth2BasicSecretEphemeralResource)(nil)
	_ ephemeral.EphemeralResourceWithConfigure = (*oauth2BasicSecretEphemeralResource)(nil)
)

func NewOAuth2BasicSecretEphemeralResource() ephemeral.EphemeralResource {
	return &oauth2BasicSecretEphemeralResource{}
}

type oauth2BasicSecretEphemeralResource struct {
	client *client.Client
}

type oauth2BasicSecretModel struct {
	Client       types.String `tfsdk:"client"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (r *oauth2BasicSecretEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth2_basic_secret"
}

func (r *oauth2BasicSecretEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Reads a Kanidm OAuth2 basic client secret during a run.

The secret is not written to the plan or state. Pass it to a write-only argument, such as an OpenBao KV secret.

Pass the basic client's ` + "`id`" + `, not its name. On the run that creates the client, ` + "`id`" + ` is unknown at plan time, so OpenTofu defers the read until apply, after the client exists. Passing the name makes the read run at plan and fail.

The provider identity must be a member of ` + "`idm_oauth2_admins`" + `.

## Example Usage

` + "```hcl" + `
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
` + "```" + ``,
		Attributes: map[string]schema.Attribute{
			"client": schema.StringAttribute{
				MarkdownDescription: "Name or UUID of the OAuth2 basic client. Pass the client's `id` when creating the client in the same run.",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client ID, which is the client's name.",
				Computed:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret. Available only during this run.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *oauth2BasicSecretEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	data := configureEphemeral(req, resp)
	if data == nil {
		return
	}
	r.client = data.client
}

func (r *oauth2BasicSecretEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config oauth2BasicSecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identifier := config.Client.ValueString()
	oauth2Client, err := r.client.ResolveOAuth2Client(ctx, identifier)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError(
				"OAuth2 Client Not Found",
				fmt.Sprintf("OAuth2 client %q was not found.", identifier),
			)
			return
		}
		resp.Diagnostics.AddError("Error Reading OAuth2 Client", err.Error())
		return
	}
	if oauth2Client.IsPublic {
		resp.Diagnostics.AddError(
			"Public OAuth2 Client",
			fmt.Sprintf("OAuth2 client %q is a public client and has no client secret.", identifier),
		)
		return
	}

	secret, err := r.client.GetOAuth2BasicSecret(ctx, oauth2Client.Name)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError(
				"OAuth2 Client Not Found",
				fmt.Sprintf("OAuth2 client %q was not found.", identifier),
			)
			return
		}
		if errors.Is(err, client.ErrForbidden) {
			resp.Diagnostics.AddError(
				"Permission Denied",
				"Reading an OAuth2 basic client secret requires the provider identity to be a member of idm_oauth2_admins.",
			)
			return
		}
		resp.Diagnostics.AddError("Error Reading OAuth2 Client Secret", err.Error())
		return
	}

	config.ClientID = types.StringValue(oauth2Client.Name)
	config.ClientSecret = types.StringValue(secret)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &config)...)
}
