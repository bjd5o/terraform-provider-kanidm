package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ssoriche/terraform-provider-kanidm/internal/client"
)

func testAccProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"kanidm": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("KANIDM_URL") == "" {
		t.Fatal("KANIDM_URL must be set for acceptance tests")
	}
	if os.Getenv("KANIDM_TOKEN") == "" {
		t.Fatal("KANIDM_TOKEN must be set for acceptance tests")
	}
}

func testAccProviderConfig() string {
	insecure := os.Getenv("KANIDM_INSECURE_SKIP_VERIFY") == "1" || os.Getenv("KANIDM_INSECURE_SKIP_VERIFY") == "true"
	return fmt.Sprintf(`
provider "kanidm" {
  url                  = %q
  token                = %q
  insecure_skip_verify = %t
}
`, os.Getenv("KANIDM_URL"), os.Getenv("KANIDM_TOKEN"), insecure)
}

func testAccClient() *client.Client {
	opts := []client.ClientOption{}
	if os.Getenv("KANIDM_INSECURE_SKIP_VERIFY") == "1" || os.Getenv("KANIDM_INSECURE_SKIP_VERIFY") == "true" {
		opts = append(opts, client.WithInsecureSkipVerify())
	}
	return client.NewClient(os.Getenv("KANIDM_URL"), os.Getenv("KANIDM_TOKEN"), opts...)
}

func TestAccServiceAccountNoToken(t *testing.T) {
	name := acctest.RandomWithPrefix("tfsa")
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "kanidm_service_account" "test" {
  name             = %q
  displayname      = "Acceptance"
  entry_managed_by = "idm_admins"
}
`, name)
	updated := testAccProviderConfig() + fmt.Sprintf(`
resource "kanidm_service_account" "test" {
  name             = %q
  displayname      = "Acceptance updated"
  entry_managed_by = "idm_admins"
}
`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kanidm_service_account.test", "id"),
					resource.TestCheckNoResourceAttr("kanidm_service_account.test", "api_token"),
					resource.TestCheckNoResourceAttr("kanidm_service_account.test", "generate_api_token"),
					checkNoServiceAccountTokens("kanidm_service_account.test"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_service_account.test", "displayname", "Acceptance updated"),
					resource.TestCheckNoResourceAttr("kanidm_service_account.test", "api_token"),
					checkNoServiceAccountTokens("kanidm_service_account.test"),
				),
			},
			{
				ResourceName:            "kanidm_service_account.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"entry_managed_by"},
			},
		},
	})
}

func TestAccPersonPasswordWriteOnly(t *testing.T) {
	name := acctest.RandomWithPrefix("tfperson")
	first := "first-password-value"
	second := "second-password-value"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: personConfig(name, first, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_person.test", "password_wo_version", "1"),
					resource.TestCheckNoResourceAttr("kanidm_person.test", "password"),
					resource.TestCheckNoResourceAttr("kanidm_person.test", "password_wo"),
					resource.TestCheckNoResourceAttr("kanidm_person.test", "initial_credential_reset_token"),
					stateOmits(first),
					checkPassword(name, first),
				),
			},
			{
				Config:             personConfig(name, second, 1),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: personConfig(name, second, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_person.test", "password_wo_version", "2"),
					stateOmits(second),
					checkPassword(name, second),
					checkPasswordRejected(name, first),
				),
			},
		},
	})
}

func TestAccOAuth2BasicNoSecret(t *testing.T) {
	name := acctest.RandomWithPrefix("tfoauth")
	config := oauth2Config(name, "Grafana")
	updated := oauth2Config(name, "Grafana updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kanidm_oauth2_basic.test", "id"),
					resource.TestCheckNoResourceAttr("kanidm_oauth2_basic.test", "client_secret"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("kanidm_oauth2_basic.test", "client_secret"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_oauth2_basic.test", "displayname", "Grafana updated"),
					resource.TestCheckNoResourceAttr("kanidm_oauth2_basic.test", "client_secret"),
				),
			},
			{
				ResourceName:      "kanidm_oauth2_basic.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOAuth2BasicSecretEphemeral(t *testing.T) {
	clientName := acctest.RandomWithPrefix("tfsecret")
	personName := acctest.RandomWithPrefix("tfsecretuser")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "kanidm_oauth2_basic" "test" {
  name        = %q
  displayname = "Secret"
  origin      = "https://app.example.com"
}

ephemeral "kanidm_oauth2_basic_secret" "test" {
  client = kanidm_oauth2_basic.test.id
}

resource "kanidm_person" "test" {
  name                = %q
  displayname         = "Secret reader"
  password_wo         = ephemeral.kanidm_oauth2_basic_secret.test.client_secret
  password_wo_version = 1
}
`, clientName, personName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("kanidm_oauth2_basic.test", "client_secret"),
					resource.TestCheckNoResourceAttr("kanidm_person.test", "password_wo"),
					checkEphemeralSecretMatchesLogin(clientName, personName),
				),
			},
		},
	})
}

func personConfig(name, password string, version int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "kanidm_person" "test" {
  name                = %q
  displayname         = "Acceptance"
  password_wo         = %q
  password_wo_version = %d
}
`, name, password, version)
}

func oauth2Config(name, displayName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "kanidm_oauth2_basic" "test" {
  name        = %q
  displayname = %q
  origin      = "https://grafana.example.com"
}
`, name, displayName)
}

func checkNoServiceAccountTokens(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := resourceID(s, resourceName)
		if err != nil {
			return err
		}
		tokens, err := testAccClient().ListServiceAccountAPITokens(context.Background(), id)
		if err != nil {
			return err
		}
		if len(tokens) != 0 {
			return fmt.Errorf("service account %s has %d API tokens", id, len(tokens))
		}
		return nil
	}
}

func checkPassword(username, password string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		return testAccClient().AuthenticatePassword(context.Background(), username, password)
	}
}

func checkPasswordRejected(username, password string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		err := testAccClient().AuthenticatePassword(context.Background(), username, password)
		if err == nil {
			return fmt.Errorf("password for %s was accepted after it should have changed", username)
		}
		return nil
	}
}

func checkEphemeralSecretMatchesLogin(clientName, personName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		secret, err := testAccClient().GetOAuth2BasicSecret(context.Background(), clientName)
		if err != nil {
			return err
		}
		if err := stateOmits(secret)(s); err != nil {
			return err
		}
		return testAccClient().AuthenticatePassword(context.Background(), personName, secret)
	}
}

func stateOmits(secret string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Primary == nil {
				continue
			}
			for key, value := range rs.Primary.Attributes {
				if strings.Contains(value, secret) {
					return fmt.Errorf("state attribute %s.%s contains a secret", rs.Type, key)
				}
			}
		}
		return nil
	}
}

func resourceID(s *terraform.State, name string) (string, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok || rs.Primary == nil || rs.Primary.Attributes["id"] == "" {
		return "", fmt.Errorf("resource %s has no id", name)
	}
	return rs.Primary.Attributes["id"], nil
}
