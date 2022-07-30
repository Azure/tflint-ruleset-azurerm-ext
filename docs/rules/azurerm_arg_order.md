# azurerm_arg_order

Recommend proper argument order within azurerm provider/resource/data blocks
The arguments are split into the following types:
head-meta (for-each/count, provider), required-az, optional-az, non-az-or-meta (the arguments of dynamic block and for future use), tail-meta (dynamic, lifecycle, depends_on)
The arguments with different types would be sorted in the order above and split by a brank line, 
while the arguments with the same type would be sorted in alphabetic order.

## Example

```hcl
resource "azurerm_container_group" "example" {
  name                = "example-continst"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  ip_address_type     = "Public"
  dns_name_label      = "aci-label"
  os_type             = "Linux"
  depends_on          = [
    azurerm_resource_group.example
  ]

  container {
    name   = "hello-world"
    image  = "mcr.microsoft.com/azuredocs/aci-helloworld:latest"
    cpu    = "0.5"
    memory = "1.5"

    ports {
          port     = 443
          protocol = "TCP"
    }
  }

  container {
    name   = "sidecar"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    cpu    = "0.5"
    memory = "1.5"
  }

  tags = {
    environment = "testing"
  }
}
```

```
$ tflint
1 issue(s) found:

Notice: Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-helloworld:latest"
    memory = "1.5"
    name   = "hello-world"

    ports {
      port     = 443
      protocol = "TCP"
    }
  }
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name

  dns_name_label  = "aci-label"
  ip_address_type = "Public"
  tags = {
    environment = "testing"
  }

  depends_on = [
    azurerm_resource_group.example
  ]
} (azurerm_arg_order)

  on main.tf line 1:
   1: resource "azurerm_container_group" "example" {

Reference: https://github.com/terraform-linters/tflint-ruleset-azurerm-ext/blob/v0.0.1/docs/rules/azurerm_arg_order.md
```

## Why

It helps to improve the readability of terraform code by splitting different types of arguments and arrange the same type of them in alphabetic order. 

## How To Fix

Just copy the text with recommended argument order of a specific block and paste it in the tf config file to overwrite the original style of this block.