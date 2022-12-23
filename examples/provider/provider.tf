terraform {
  required_providers {
    xelon = {
      source  = "Xelon-AG/xelon"
      version = ">= 0.1.0"
    }
  }
}

# Set the variable value in *.tfvars file
# or using -var="xelon_token=..." CLI option
variable "xelon_token" {}

# Configure the Xelon Provider
provider "xelon" {
  token = var.xelon_token
}
