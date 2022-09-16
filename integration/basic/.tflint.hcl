config {
  disabled_by_default = true
}

plugin "azurerm-ext" {
  enabled = true
}

rule "azurerm_arg_order" {
    enabled = true
}

rule "terraform_required_providers" {
  enabled = false
}

rule "terraform_required_version" {
  enabled = false
}