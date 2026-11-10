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

# Example: Person account that sets up its own credentials.
# Kanidm queues a credential reset token message for the primary mail address
# (the first entry) when the person is created, and queues another each time
# credentials_version changes. A mail sender such as kanidm-mail-sender
# delivers it later. OpenTofu does not check delivery, so if the message does
# not arrive, change credentials_version again. The token is not stored in the
# plan or state.
resource "kanidm_person" "charlie" {
  name                = "charlie"
  displayname         = "Charlie Brown"
  mail                = ["charlie@example.com", "cbrown@example.com"]
  credentials_version = 1
}

# Example: Imported existing person account
# Import command: terraform import kanidm_person.existing_user username
resource "kanidm_person" "existing_user" {
  name        = "existing"
  displayname = "Existing User"
}
