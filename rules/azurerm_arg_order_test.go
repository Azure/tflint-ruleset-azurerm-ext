package rules

import (
	"testing"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_AzurermArgOrderRule(t *testing.T) {

	cases := []struct {
		Name     string
		Content  string
		Expected helper.Issues
	}{
		{
			Name: "1. attr alphabetic order",
			Content: `
resource "azurerm_container_group" "example" {
  os_type             = "Linux"
  name                = "example-continst"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 45,
						},
					},
				},
			},
		},

		{
			Name: "2. split and reorder of required and optional arg",
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
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
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
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 45,
						},
					},
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
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
ports {
  port     = 443
  protocol = "TCP"
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   7,
							Column: 5,
						},
						End: hcl.Pos{
							Line:   7,
							Column: 10,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
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
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   3,
							Column: 3,
						},
						End: hcl.Pos{
							Line:   3,
							Column: 12,
						},
					},
				},
			},
		},

		{
			Name: "4. Meta Arg",
			Content: `
resource "azurerm_virtual_network" "vnet" {
  address_space       = ["10.0.0.0/16"]
  count               = 4
  depends_on = [
    azurerm_resource_group.example
  ]
  location            = azurerm_resource_group.example.location
  name                = "myTFVnet"
  resource_group_name = azurerm_resource_group.rg.name

  tags = {
    Name = "VM network ${count.index}"
  }
}

resource "azurerm_virtual_network" "vnet" {
  address_space       = ["10.0.0.0/16"]
  depends_on = [
    azurerm_resource_group.example
  ]
  for_each            = local.subnet_ids
  location            = azurerm_resource_group.example.location
  name                = "myTFVnet"
  resource_group_name = azurerm_resource_group.rg.name

  tags = {
    Name = "VM network ${each.key}"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "azurerm_virtual_network" "vnet" {
  count = 4
  
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.example.location
  name                = "myTFVnet"
  resource_group_name = azurerm_resource_group.rg.name

  tags = {
    Name = "VM network ${count.index}"
  }
  
  depends_on = [
    azurerm_resource_group.example
  ]
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 42,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "azurerm_virtual_network" "vnet" {
  for_each = local.subnet_ids

  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.example.location
  name                = "myTFVnet"
  resource_group_name = azurerm_resource_group.rg.name

  tags = {
    Name = "VM network ${each.key}"
  }

  depends_on = [
    azurerm_resource_group.example
  ]
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   17,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   17,
							Column: 42,
						},
					},
				},
			},
		},

		{
			Name: "5. dynamic block",
			Content: `
resource "azurerm_container_group" "example" {
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name

  dynamic "container" {
    content {
      name   = container.value["name"]
      image  = container.value["image"]
      cpu 	 = container.value["cpu"]
      ports {
	    port     = 443
	    protocol = "TCP"
      }
      memory = container.value["memory"]
    }
	for_each = var.containers
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
content {
  cpu 	 = container.value["cpu"]
  image  = container.value["image"]
  memory = container.value["memory"]
  name   = container.value["name"]

  ports {
	port     = 443
	protocol = "TCP"
  }
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   9,
							Column: 5,
						},
						End: hcl.Pos{
							Line:   9,
							Column: 12,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
dynamic "container" {
  for_each = var.containers

  content {
    cpu 	 = container.value["cpu"]
    image  = container.value["image"]
    memory = container.value["memory"]
    name   = container.value["name"]

    ports {
      port     = 443
	  protocol = "TCP"
    }
  }
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   8,
							Column: 3,
						},
						End: hcl.Pos{
							Line:   8,
							Column: 22,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "azurerm_container_group" "example" {
  dynamic "container" {
    for_each = var.containers

    content {
      cpu 	 = container.value["cpu"]
      image  = container.value["image"]
      memory = container.value["memory"]
      name   = container.value["name"]

      ports {
        port     = 443
	    protocol = "TCP"
      }
    }
  }
  location            = azurerm_resource_group.example.location
  name                = "example-continst"
  os_type             = "Linux"
  resource_group_name = azurerm_resource_group.example.name
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 45,
						},
					},
				},
			},
		},

		{
			Name: "6. common",
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
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "azurerm_resource_group" "example" {
  location = "West Europe"
  name     = "example-resources"
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 44,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
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
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   15,
							Column: 3,
						},
						End: hcl.Pos{
							Line:   15,
							Column: 12,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
container {
  cpu    = "0.5"
  image  = "mcr.microsoft.com/azuredocs/aci-tutorial-sidecar"
  memory = "1.5"
  name   = "sidecar"
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   27,
							Column: 3,
						},
						End: hcl.Pos{
							Line:   27,
							Column: 12,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
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

  dns_name_label      = "aci-label"
  ip_address_type     = "Public"
  tags                = {
    environment = "testing"
  }
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   7,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   7,
							Column: 45,
						},
					},
				},
			},
		},
		{
			Name: "7. datasource",
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
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
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
			Name: "8. provider",
			Content: `
provider "azurerm" {
  client_id   = "temp"
  features {}
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
provider "azurerm" {
  features {}

  client_id   = "temp"
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   2,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 19,
						},
					},
				},
			},
		},
	}

	rule := NewAzurermArgOrderRule()

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
