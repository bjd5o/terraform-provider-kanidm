package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestResourceSchemasHaveNoStoredSecrets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resources := []resource.Resource{
		NewPersonResource(),
		NewServiceAccountResource(),
		NewApplicationResource(),
		NewGroupResource(),
		NewGroupMembersResource(),
		NewAccountPolicyResource(),
		NewSystemDeniedNamesResource(),
		NewOAuth2BasicResource(),
		NewOAuth2PublicResource(),
	}

	for _, res := range resources {
		var meta resource.MetadataResponse
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "kanidm"}, &meta)

		var resp resource.SchemaResponse
		res.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s schema diagnostics: %v", meta.TypeName, resp.Diagnostics)
		}

		walkAttributes(t, meta.TypeName, resp.Schema.Attributes)
		for name, block := range resp.Schema.Blocks {
			walkBlock(t, meta.TypeName+"."+name, block)
		}
	}
}

func TestPersonPasswordIsWriteOnly(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewPersonResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["password_wo"]
	if !ok {
		t.Fatal("password_wo is missing")
	}
	if !attr.IsSensitive() || !attr.IsWriteOnly() {
		t.Fatal("password_wo must be sensitive and write-only")
	}
	if _, ok := resp.Schema.Attributes["password"]; ok {
		t.Fatal("password is still in the schema")
	}
}

func walkAttributes(t *testing.T, prefix string, attrs map[string]schema.Attribute) {
	t.Helper()

	for name, attr := range attrs {
		path := prefix + "." + name
		if attr.IsSensitive() && !attr.IsWriteOnly() {
			t.Errorf("%s is Sensitive without WriteOnly, so it would be stored in state", path)
		}
		if nested, ok := attr.(schema.NestedAttribute); ok {
			obj, ok := nested.GetNestedObject().(schema.NestedAttributeObject)
			if !ok {
				t.Fatalf("%s nested object has type %T", path, nested.GetNestedObject())
			}
			walkAttributes(t, path, obj.Attributes)
		}
	}
}

func walkBlock(t *testing.T, prefix string, block schema.Block) {
	t.Helper()

	obj, ok := block.GetNestedObject().(schema.NestedBlockObject)
	if !ok {
		t.Fatalf("%s nested object has type %T", prefix, block.GetNestedObject())
	}
	walkAttributes(t, prefix, obj.Attributes)
	for name, nested := range obj.Blocks {
		walkBlock(t, prefix+"."+name, nested)
	}
}
