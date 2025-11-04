terraform {
  required_providers {
    xelon = {
      source = "xelon-ag/xelon"
    }
  }
}

provider "xelon" {
  # Configuration will be taken from environment variables:
  # - XELON_TOKEN
  # - XELON_CLIENT_ID (optional)
}
