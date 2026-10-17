package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestServiceAccountUpgradeDropsToken(t *testing.T) {
	t.Parallel()

	const token = "terraform-managed-token-secret"
	managers, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"openbao-token-issuers"})
	if diags.HasError() {
		t.Fatal(diags)
	}

	upgraded := upgradeResourceState(t, NewServiceAccountResource(), 0, struct {
		Name             types.String `tfsdk:"name"`
		ID               types.String `tfsdk:"id"`
		DisplayName      types.String `tfsdk:"displayname"`
		Mail             types.List   `tfsdk:"mail"`
		PosixEnabled     types.Bool   `tfsdk:"posix_enabled"`
		GIDNumber        types.Int64  `tfsdk:"gidnumber"`
		Shell            types.String `tfsdk:"shell"`
		GenerateAPIToken types.Bool   `tfsdk:"generate_api_token"`
		APIToken         types.String `tfsdk:"api_token"`
		EntryManagedBy   types.Set    `tfsdk:"entry_managed_by"`
		ValidFrom        types.String `tfsdk:"valid_from"`
		ExpireAt         types.String `tfsdk:"expire_at"`
	}{
		Name:             types.StringValue("mailer"),
		ID:               types.StringValue("uuid-1"),
		DisplayName:      types.StringValue("Mailer"),
		Mail:             types.ListNull(types.StringType),
		PosixEnabled:     types.BoolValue(false),
		GenerateAPIToken: types.BoolValue(true),
		APIToken:         types.StringValue(token),
		EntryManagedBy:   managers,
	})

	assertStateOmits(t, upgraded, token, "api_token", "generate_api_token")

	var model serviceAccountResourceModel
	if diags := upgraded.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	if model.Name.ValueString() != "mailer" {
		t.Fatalf("name = %q", model.Name.ValueString())
	}
	if model.EntryManagedBy.ValueString() != "openbao-token-issuers" {
		t.Fatalf("entry_managed_by = %q", model.EntryManagedBy.ValueString())
	}
}

func TestServiceAccountUpgradeFromV1DropsToken(t *testing.T) {
	t.Parallel()

	const token = "still-in-v1-state"
	upgraded := upgradeResourceState(t, NewServiceAccountResource(), 1, struct {
		Name             types.String `tfsdk:"name"`
		ID               types.String `tfsdk:"id"`
		DisplayName      types.String `tfsdk:"displayname"`
		Mail             types.List   `tfsdk:"mail"`
		PosixEnabled     types.Bool   `tfsdk:"posix_enabled"`
		GIDNumber        types.Int64  `tfsdk:"gidnumber"`
		Shell            types.String `tfsdk:"shell"`
		GenerateAPIToken types.Bool   `tfsdk:"generate_api_token"`
		APIToken         types.String `tfsdk:"api_token"`
		EntryManagedBy   types.String `tfsdk:"entry_managed_by"`
		ValidFrom        types.String `tfsdk:"valid_from"`
		ExpireAt         types.String `tfsdk:"expire_at"`
	}{
		Name:             types.StringValue("mailer"),
		ID:               types.StringValue("uuid-1"),
		DisplayName:      types.StringValue("Mailer"),
		Mail:             types.ListNull(types.StringType),
		PosixEnabled:     types.BoolValue(false),
		GenerateAPIToken: types.BoolValue(true),
		APIToken:         types.StringValue(token),
		EntryManagedBy:   types.StringValue("openbao-token-issuers"),
	})

	assertStateOmits(t, upgraded, token, "api_token", "generate_api_token")

	var model serviceAccountResourceModel
	if diags := upgraded.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	if model.EntryManagedBy.ValueString() != "openbao-token-issuers" {
		t.Fatalf("entry_managed_by = %q", model.EntryManagedBy.ValueString())
	}
}

func TestPersonUpgradeDropsSecrets(t *testing.T) {
	t.Parallel()

	const password = "person-password-secret"
	const resetToken = "reset-token-secret"
	upgraded := upgradeResourceState(t, NewPersonResource(), 0, struct {
		Name                                types.String `tfsdk:"name"`
		ID                                  types.String `tfsdk:"id"`
		DisplayName                         types.String `tfsdk:"displayname"`
		LegalName                           types.String `tfsdk:"legalname"`
		PosixEnabled                        types.Bool   `tfsdk:"posix_enabled"`
		GIDNumber                           types.Int64  `tfsdk:"gidnumber"`
		Shell                               types.String `tfsdk:"shell"`
		ValidFrom                           types.String `tfsdk:"valid_from"`
		ExpireAt                            types.String `tfsdk:"expire_at"`
		NameManagement                      types.String `tfsdk:"name_management"`
		DisplayManagement                   types.String `tfsdk:"display_management"`
		LegalNameManagement                 types.String `tfsdk:"legalname_management"`
		Mail                                types.List   `tfsdk:"mail"`
		Password                            types.String `tfsdk:"password"`
		GenerateInitialCredentialResetToken types.Bool   `tfsdk:"generate_initial_credential_reset_token"`
		InitialCredentialResetToken         types.String `tfsdk:"initial_credential_reset_token"`
		InitialCredentialResetTokenTTL      types.Int64  `tfsdk:"initial_credential_reset_token_ttl"`
	}{
		Name:                                types.StringValue("alice"),
		ID:                                  types.StringValue("uuid-alice"),
		DisplayName:                         types.StringValue("Alice"),
		Mail:                                types.ListNull(types.StringType),
		PosixEnabled:                        types.BoolValue(false),
		Password:                            types.StringValue(password),
		GenerateInitialCredentialResetToken: types.BoolValue(true),
		InitialCredentialResetToken:         types.StringValue(resetToken),
		InitialCredentialResetTokenTTL:      types.Int64Value(3600),
	})

	assertStateOmits(t, upgraded, password, "password", "generate_initial_credential_reset_token", "initial_credential_reset_token", "initial_credential_reset_token_ttl")
	assertStateOmits(t, upgraded, resetToken)

	var model personResourceModel
	if diags := upgraded.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	if !model.PasswordWOVersion.IsNull() {
		t.Fatalf("password_wo_version = %#v, want null", model.PasswordWOVersion)
	}
	if model.Name.ValueString() != "alice" {
		t.Fatalf("name = %q", model.Name.ValueString())
	}
}

func TestOAuth2BasicUpgradeDropsSecret(t *testing.T) {
	t.Parallel()

	const secret = "oauth-client-secret"
	upgraded := upgradeResourceState(t, NewOAuth2BasicResource(), 0, struct {
		ID                       types.String `tfsdk:"id"`
		Name                     types.String `tfsdk:"name"`
		DisplayName              types.String `tfsdk:"displayname"`
		Origin                   types.String `tfsdk:"origin"`
		RedirectURIs             types.Set    `tfsdk:"redirect_uris"`
		ImagePath                types.String `tfsdk:"image_path"`
		ImageSHA256              types.String `tfsdk:"image_sha256"`
		ScopeMaps                types.Set    `tfsdk:"scope_map"`
		SupScopeMaps             types.Set    `tfsdk:"sup_scope_map"`
		ClaimMaps                types.Set    `tfsdk:"claim_map"`
		PreferShortUsername      types.Bool   `tfsdk:"prefer_short_username"`
		AllowInsecureDisablePKCE types.Bool   `tfsdk:"allow_insecure_client_disable_pkce"`
		JWTLegacyCryptoEnable    types.Bool   `tfsdk:"jwt_legacy_crypto_enable"`
		ClientSecret             types.String `tfsdk:"client_secret"`
	}{
		ID:           types.StringValue("uuid-grafana"),
		Name:         types.StringValue("grafana"),
		DisplayName:  types.StringValue("Grafana"),
		Origin:       types.StringValue("https://grafana.example.com"),
		RedirectURIs: types.SetNull(types.StringType),
		ScopeMaps:    types.SetNull(oauth2ScopeMapType),
		SupScopeMaps: types.SetNull(oauth2ScopeMapType),
		ClaimMaps:    types.SetNull(oauth2ClaimMapType),
		ClientSecret: types.StringValue(secret),
	})

	assertStateOmits(t, upgraded, secret, "client_secret")

	var model oauth2BasicResourceModel
	if diags := upgraded.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	if model.Name.ValueString() != "grafana" {
		t.Fatalf("name = %q", model.Name.ValueString())
	}
}

func TestOAuth2PublicSchemaStaysAtVersionZero(t *testing.T) {
	t.Parallel()

	basic := resourceSchema(t, NewOAuth2BasicResource())
	public := resourceSchema(t, NewOAuth2PublicResource())
	if basic.Version != 1 {
		t.Fatalf("basic schema version = %d, want 1", basic.Version)
	}
	if public.Version != 0 {
		t.Fatalf("public schema version = %d, want 0", public.Version)
	}
	if _, ok := public.Attributes["client_secret"]; ok {
		t.Fatal("public schema has client_secret")
	}
	if upgraders := NewOAuth2PublicResource().(resource.ResourceWithUpgradeState).UpgradeState(context.Background()); len(upgraders) != 0 {
		t.Fatalf("public upgraders = %d, want 0", len(upgraders))
	}
}

func upgradeResourceState(t *testing.T, res resource.Resource, from int64, prior any) tfsdk.State {
	t.Helper()

	ctx := context.Background()
	upgrader, ok := res.(resource.ResourceWithUpgradeState)
	if !ok {
		t.Fatalf("%T does not upgrade state", res)
	}
	upgraders := upgrader.UpgradeState(ctx)
	stateUpgrader, ok := upgraders[from]
	if !ok {
		t.Fatalf("no upgrader from version %d", from)
	}

	current := resourceSchema(t, res)
	priorState := tfsdk.State{
		Schema: *stateUpgrader.PriorSchema,
		Raw:    tftypes.NewValue(stateUpgrader.PriorSchema.Type().TerraformType(ctx), nil),
	}
	if diags := priorState.Set(ctx, prior); diags.HasError() {
		t.Fatal(diags)
	}

	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{
			Schema: current,
			Raw:    tftypes.NewValue(current.Type().TerraformType(ctx), nil),
		},
	}
	stateUpgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &priorState}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

var (
	oauth2ScopeMapType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"group":  types.StringType,
		"scopes": types.SetType{ElemType: types.StringType},
	}}
	oauth2ClaimMapType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":   types.StringType,
		"group":  types.StringType,
		"values": types.SetType{ElemType: types.StringType},
		"join":   types.StringType,
	}}
)

func resourceSchema(t *testing.T, res resource.Resource) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}

func assertStateOmits(t *testing.T, state tfsdk.State, secret string, attrs ...string) {
	t.Helper()

	rendered := state.Raw.String()
	if secret != "" && strings.Contains(rendered, secret) {
		t.Fatalf("upgraded state still contains %q: %s", secret, rendered)
	}
	for _, attr := range attrs {
		if strings.Contains(rendered, `"`+attr+`"`) {
			t.Fatalf("upgraded state still contains attribute %s: %s", attr, rendered)
		}
	}
}
