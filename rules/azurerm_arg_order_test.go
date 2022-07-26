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
			Name: "Meta Arg",
			Content: `
resource "aws_instance" "server" {
  ami           = "ami-a1b2c3d4"
  count = 4 # create four similar EC2 instances
  depends_on = [
    aws_iam_role_policy.example
  ]
  iam_instance_profile = aws_iam_instance_profile.example
  instance_type = "t2.micro"
  tags = {
    Name = "Server ${count.index}"
  }
}

resource "aws_instance" "server" {
  ami           = "ami-a1b2c3d4"
  depends_on = [
    aws_iam_role_policy.example
  ]
  iam_instance_profile = aws_iam_instance_profile.example
  instance_type = "t2.micro"
  subnet_id     = each.key # note: each.key and each.value are the same for a set
  for_each = local.subnet_ids
  
  tags = {
    Name = "Server ${each.key}"
  }
}`,
			Expected: helper.Issues{
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "aws_instance" "server" {
  count                = 4
  ami                  = "ami-a1b2c3d4"
  iam_instance_profile = aws_iam_instance_profile.example
  instance_type        = "t2.micro"
  tags                 = {
    Name = "Server ${count.index}"
  }
  depends_on           = [
    aws_iam_role_policy.example
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
							Column: 33,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
resource "aws_instance" "server" {
  for_each             = local.subnet_ids
  ami                  = "ami-a1b2c3d4"
  iam_instance_profile = aws_iam_instance_profile.example
  instance_type        = "t2.micro"
  subnet_id            = each.key
  tags                 = {
    Name = "Server ${each.key}"
  }
  depends_on           = [
    aws_iam_role_policy.example
  ]
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   15,
							Column: 1,
						},
						End: hcl.Pos{
							Line:   15,
							Column: 33,
						},
					},
				},
			},
		},
		{
			Name: "dynamic block",
			Content: `
resource "azurerm_container_group" "example" {
  location = "West Europe"
  name     = "example-resources"

  dynamic "setting" {
    content {
      namespace = setting.value["namespace"]
      name 		= setting.value["name"]
      value 	= setting.value["value"]
    }
	for_each = var.settings
  }
}`,
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
							Line:   7,
							Column: 5,
						},
						End: hcl.Pos{
							Line:   7,
							Column: 12,
						},
					},
				},
				{
					Rule: NewAzurermArgOrderRule(),
					Message: `Arguments are not sorted in azurerm doc order, correct order is:
dynamic "setting" {
  for_each = var.settings
  content {
    name      = setting.value["name"]
    namespace = setting.value["namespace"]
    value     = setting.value["value"]
  }
}`,
					Range: hcl.Range{
						Filename: "config.tf",
						Start: hcl.Pos{
							Line:   6,
							Column: 3,
						},
						End: hcl.Pos{
							Line:   6,
							Column: 20,
						},
					},
				},
			},
		},
		{
			Name: "common",
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
