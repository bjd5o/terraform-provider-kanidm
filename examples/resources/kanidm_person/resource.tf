# Example: Person account with a write-only initial password.
# password_wo is sent to Kanidm and is not stored in the plan or state.
# Increase password_wo_version to change the password.
resource "kanidm_person" "alice" {
  name                = "alice"
  displayname         = "Alice Smith"
  mail                = ["alice@example.com"]
  password_wo         = ephemeral.vault_generic_secret.alice.data.password
  password_wo_version = 1
}

# Example: Person account without a password.
# Credential reset tokens are minted outside OpenTofu, for example by the mailer.
resource "kanidm_person" "charlie" {
  name        = "charlie"
  displayname = "Charlie Brown"
  mail        = ["charlie@example.com", "cbrown@example.com"]
}

# Example: Imported existing person account
# Import command: terraform import kanidm_person.existing_user username
resource "kanidm_person" "existing_user" {
  name        = "existing"
  displayname = "Existing User"
}
