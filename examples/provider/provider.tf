terraform {
  required_providers {
    xelon = {
      source  = "Xelon-AG/xelon"
      version = ">= 1.0.0"
    }
  }
}

# Set the variable value in *.tfvars file
# or using -var="xelon_client_id=..." CLI option
variable "xelon_client_id" {}

# Set the variable value in *.tfvars file
# or using -var="xelon_token=..." CLI option
variable "xelon_token" {}

# Configure the Xelon Provider
provider "xelon" {
  client_id = var.xelon_client_id
  token     = var.xelon_token
}
