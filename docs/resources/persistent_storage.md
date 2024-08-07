---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "xelon_persistent_storage Resource - terraform-provider-xelon"
subcategory: ""
description: |-
  The persistent storage resource allows you to manage Xelon Persistent Storages.
---

# xelon_persistent_storage (Resource)

The persistent storage resource allows you to manage Xelon Persistent Storages.

## Example Usage

```terraform
resource "xelon_persistent_storage" "backup" {
  cloud_id = data.xelon_cloud.hcp.cloud_id
  name     = "backup"
  size     = 50
}

data "xelon_cloud" "hcp" {
  name = "Main HCP Cloud"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cloud_id` (Number) The ID of the organization cloud.
- `name` (String) The name of the persistent storage.
- `size` (Number) The size of the persistent storage in GB. If updated, can only be expanded.

### Read-Only

- `formatted` (Boolean) True, if the persistent storage is formatted.
- `id` (String) The ID of this resource.
- `uuid` (String) The UUID of the persistent storage.
