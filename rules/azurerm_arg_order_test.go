package rules

import (
	"testing"

	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_AzurermArgOrderRule(t *testing.T) {

	cases := []struct {
		Name     string
		Content  string
		Expected helper.Issues
	}{
		{
			Name: "1. sorting args in alphabetic order",
			Content: `
resource "azurerm_container_group" "example" {
  os_type             = "Linux"
  name                = "example-continst"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name

  dns_config {
    nameservers = []
  }
  diagnostics {
    log_analytics {
      workspace_id  = "test"
      workspace_key = "test"
    }
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name

  diagnostics {
    log_analytics {
      workspace_id  = "test"
      workspace_key = "test"
    }
  }
  dns_config {
    nameservers = []
  }
}`,
				},
			},
		},
		{
			Name: "2. reorder of required and optional args",
			Content: `
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  ip_address_type     = "Public"
  dns_name_label      = "aci-label"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  tags                = {
    environment = "testing"
  }
  
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
  diagnostics {
    log_analytics {
      workspace_id  = "test"
      workspace_key = "test"
    }
  }
  dns_config {
    nameservers = []
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags                = {
    environment = "testing"
  }

  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
  diagnostics {
    log_analytics {
      workspace_id  = "test"
      workspace_key = "test"
    }
  }
  dns_config {
    nameservers = []
  }
}`,
				},
			},
		},
		{
			Name: "3. reorder of args in nested block",
			Content: `
resource "azurerm_container_group" "example" {
  container {
    name   = "hello-world"
    image  = "mcr.microsoft.com/azuredocs/aci-helloworld:latest"
    cpu    = "0.5"
    ports {
      protocol = "TCP"
	  port     = 443
    }
    memory = "1.5"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
container {
  cpu    = "0.5"
  image  = "mcr.microsoft.com/azuredocs/aci-helloworld:latest"
  memory = "1.5"
  name   = "hello-world"

  ports {
    port     = 443
    protocol = "TCP"
  }
}`,
				},
			},
		},

		{
			Name: "4. Meta Arg",
			Content: `
resource "azurerm_container_group" "example" {

  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  count               = 4
  os_type             = "Linux"
  provider            = azurerm.europe
  resource_group_name = azurerm_resource_group.example.name

  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags                = {
    Name = "container ${count.index}"
  }
  depends_on = [
    azurerm_resource_group.example
  ]

  lifecycle {
    create_before_destroy = true
  }
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
}

resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  for_each            = local.container_ids
  os_type             = "Linux"
  provider            = azurerm.europe
  resource_group_name = azurerm_resource_group.example.name

  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  depends_on = [
    azurerm_resource_group.example
  ]
  tags = {
    Name = "container ${each.key}"
  }
  lifecycle {
    create_before_destroy = true
  }
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  count    = 4
  provider = azurerm.europe

  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label  = "aci-label"
  ip_address_type = "Public"
  tags = {
    Name = "container ${count.index}"
  }
  
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }

  lifecycle {
    create_before_destroy = true
  }
  depends_on = [
    azurerm_resource_group.example
  ]
}`,
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  for_each = local.container_ids
  provider = azurerm.europe

  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label  = "aci-label"
  ip_address_type = "Public"
  tags = {
    Name = "container ${each.key}"
  }

  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }

  lifecycle {
    create_before_destroy = true
  }
  depends_on = [
    azurerm_resource_group.example
  ]
}`,
				},
			},
		},
		{
			Name: "5. Gap between different types of args",
			Content: `
resource "azurerm_container_group" "example" {
  count               = 4
  provider            = azurerm.europe
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags = {
    Name = "container ${count.index}"
  }
  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }
  lifecycle {
    create_before_destroy = true
  }
  depends_on = [
    azurerm_resource_group.example
  ]
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  count    = 4
  provider = azurerm.europe

  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags = {
    Name = "container ${count.index}"
  }

  container {
    cpu    = "0.5"
    image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
    memory = "1.5"
    name   = "sidecar"
  }

  lifecycle {
    create_before_destroy = true
  }
  depends_on = [
    azurerm_resource_group.example
  ]
}`,
				},
			},
		},
		{
			Name: "6. dynamic block",
			Content: `
resource "azurerm_kubernetes_cluster" "main" {
  dynamic "azure_active_directory_role_based_access_control" {
    for_each = var.enable_role_based_access_control && var.rbac_aad_managed ? ["rbac"] : []
    content {
      admin_group_object_ids = var.rbac_aad_admin_group_object_ids
      azure_rbac_enabled     = var.rbac_aad_azure_rbac_enabled
      managed                = true
      tenant_id              = var.rbac_aad_tenant_id
    }
  }
  
  dynamic "ingress_application_gateway" {
    for_each = var.enable_ingress_application_gateway ? ["ingress_application_gateway"] : []

    content {
      gateway_name = var.ingress_application_gateway_name
      gateway_id   = var.ingress_application_gateway_id
      subnet_cidr  = var.ingress_application_gateway_subnet_cidr
      subnet_id    = var.ingress_application_gateway_subnet_id
    }
  }

  dynamic "identity" {
    for_each = var.client_id == "" || var.client_secret == "" ? ["identity"] : []

    content {
      type = var.identity_type
      identity_ids = var.identity_ids
    }
  }

  default_node_pool {
    name    = var.agents_pool_name
    vm_size = var.agents_size
  }

  dynamic "default_node_pool" {
    for_each = var.enable_auto_scaling == true ? [] : ["default_node_pool_manually_scaled"]

    content {
      vm_size = var.agents_size
      name    = var.agents_pool_name
    }
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_kubernetes_cluster" "main" {
  default_node_pool {
    name    = var.agents_pool_name
    vm_size = var.agents_size
  }
  dynamic "default_node_pool" {
    for_each = var.enable_auto_scaling == true ? [] : ["default_node_pool_manually_scaled"]

    content {
      name    = var.agents_pool_name
      vm_size = var.agents_size
    }
  }
  dynamic "azure_active_directory_role_based_access_control" {
    for_each = var.enable_role_based_access_control && var.rbac_aad_managed ? ["rbac"] : []

    content {
      admin_group_object_ids = var.rbac_aad_admin_group_object_ids
      azure_rbac_enabled     = var.rbac_aad_azure_rbac_enabled
      managed                = true
      tenant_id              = var.rbac_aad_tenant_id
    }
  }
  dynamic "identity" {
    for_each = var.client_id == "" || var.client_secret == "" ? ["identity"] : []

    content {
      type = var.identity_type
      identity_ids = var.identity_ids
    }
  }
  dynamic "ingress_application_gateway" {
    for_each = var.enable_ingress_application_gateway ? ["ingress_application_gateway"] : []

    content {
      gateway_id   = var.ingress_application_gateway_id
      gateway_name = var.ingress_application_gateway_name
      subnet_cidr  = var.ingress_application_gateway_subnet_cidr
      subnet_id    = var.ingress_application_gateway_subnet_id
    }
  }
}`,
				},
			},
		},

		{
			Name: "7. common",
			Content: `
resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_container_group" "example" {
  name                = "example-continst"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  ip_address_type     = "Public"
  dns_name_label      = "aci-label"
  os_type             = "Linux"

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

  tags                = {
    environment = "testing"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_resource_group" "example" {
  location = "West Europe"
  name     = "example-resources"
}`,
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags                = {
    environment = "testing"
  }
  
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
}`,
				},
			},
		},
		{
			Name: "8. datasource",
			Content: `
data "azurerm_resources" "example" {
  resource_group_name = "example-resources"
  required_tags = {
    environment = "production"
    role        = "webserver"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
data "azurerm_resources" "example" {
  required_tags = {
    environment = "production"
    role        = "webserver"
  }
  resource_group_name = "example-resources"
}`,
				},
			},
		},
		{
			Name: "9. provider",
			Content: `
provider "azurerm" {
  features {}
  client_id = "temp"
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are expected to be sorted in following order:
provider "azurerm" {
  client_id = "temp"
  
  features {}
}`,
				},
			},
		},
		{
			Name: "10. empty block",
			Content: `
resource "azurerm_container_group" "example" {}`,
			Expected: helper.Issues{},
		},
	}

	rule := NewRule(NewAzurermArgOrderRule())

	for _, tc := range cases {
		runner := helper.TestRunner(t, map[string]string{"config.tf": tc.Content})
		t.Run(tc.Name, func(t *testing.T) {
			if err := rule.Check(runner); err != nil {
				t.Fatalf("Unexpected error occurred: %s", err)
			}
			AssertIssues(t, tc.Expected, runner.Issues)
		})
	}
}
