# TFLint Ruleset for terraform-provider-azurerm
[![Build Status](https://github.com/terraform-linters/tflint-ruleset-azurerm/workflows/build/badge.svg?branch=master)](https://github.com/terraform-linters/tflint-ruleset-azurerm/actions)
[![GitHub release](https://img.shields.io/github/release/terraform-linters/tflint-ruleset-azurerm.svg)](https://github.com/terraform-linters/tflint-ruleset-azurerm/releases/latest)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-blue.svg)](LICENSE)

TFLint ruleset plugin for Terraform Provider for Azure (Resource Manager)

## Requirements

- TFLint v0.35+
- Go v1.18

## Building the plugin

Clone the repository locally and run the following command:

```
$ make
```

You can easily install the built plugin with the following:

```
$ make install
```

Note that if you install the plugin with make install, you must omit the `version` and `source` attributes in `.tflint.hcl`:

```hcl
plugin "azurerm-ext" {
    enabled = true
}
```

Follow the instructions to edit the generated files and open a new pull request.
