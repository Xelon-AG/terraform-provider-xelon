---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "xelon_device Resource - terraform-provider-xelon"
subcategory: ""
description: |-
  The device resource allows you to manage Xelon devices.
---

# xelon_device (Resource)

The device resource allows you to manage Xelon devices.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cloud_id` (Number) The cloud ID from your organization.
- `cpu_core_count` (Number) The number of CPU cores for a device
- `disk_size` (Number) Size of a disk in gigabytes.
- `display_name` (String) Display name of a device.
- `hostname` (String) Hostname of a device.
- `memory` (Number) Amount of RAM in gigabytes.
- `network` (Block List, Min: 1, Max: 1) Device network interface configuration. (see [below for nested schema](#nestedblock--network))
- `password` (String, Sensitive) Password of a device.
- `swap_disk_size` (Number) Size of a SWAP disk in gigabytes.
- `template_id` (Number) Template ID of the selected OS.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--network"></a>
### Nested Schema for `network`

Required:

- `id` (Number) Network ID available for your organization.
- `ipv4_address_id` (Number) IPv4 address ID for a device.
- `nic_controller_key` (Number) Network interface card (NIC) controller key.
- `nic_key` (Number) Network interface card (NIC) key.
- `nic_number` (Number) Network interface card (NIC) number.
- `nic_unit` (Number) Network interface card (NIC) unit.

