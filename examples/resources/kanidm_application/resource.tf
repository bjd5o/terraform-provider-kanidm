resource "kanidm_group" "mail_users" {
  name = "mail-users"
}

resource "kanidm_application" "mail" {
  name         = "mail"
  displayname  = "Mail"
  linked_group = kanidm_group.mail_users.id
}
