package rules

import (
	"testing"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_AzurermArgsOrderRule(t *testing.T) {

	cases := []struct {
		Name     string
		Content  string
		Expected helper.Issues
	}{
		{
			Name: "dynamic block",
			Content: `
resource "azurerm_container_group" "example" {
  location = "West Europe"
  name     = "example-resources"

  dynamic "setting" {
    for_each = var.settings
    content {
      namespace = setting.value["namespace"]
      name 		= setting.value["name"]
      value 	= setting.value["value"]
    }
  }
}
`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
content {
  name      = setting.value["name"]
  namespace = setting.value["namespace"]
  value     = setting.value["value"]
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   8,
							Column: 5,
						},
						End: hcl.Pos{
							Line:   8,
							Column: 12,
						},
					},
				},
			},
		},
		{
			Name: "comments",
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
	}

	rule := NewAzurermArgOrderRule()

	for _, tc := range cases {
		runner := helper.TestRunner(t, map[string]string{"config.tf": tc.Content})

		if err := rule.Check(runner); err != nil {
			t.Fatalf("Unexpected error occurred: %s", err)
		}

		helper.AssertIssues(t, tc.Expected, runner.Issues)
	}
}
