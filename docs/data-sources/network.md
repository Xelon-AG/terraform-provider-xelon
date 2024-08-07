---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "xelon_network Data Source - terraform-provider-xelon"
subcategory: ""
description: |-
  The network data source provides information about an existing network.
---

# xelon_network (Data Source)

The network data source provides information about an existing network.

## Example Usage

```terraform
data "xelon_network" "wan" {
  filter = {
    network_id = 11
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `filter` (Attributes) The filter specifies the criteria to retrieve a single network. The retrieval will fail if the criteria match more than one item. (see [below for nested schema](#nestedatt--filter))

### Read-Only

- `cloud_id` (Number) The cloud ID of the organization (tenant).
- `dns_primary` (String) The primary DNS server address.
- `dns_secondary` (String) The secondary DNS server address.
- `id` (String) The ID of the network.
- `name` (String) The name of the network.
- `netmask` (String) The netmask of the network.
- `network` (String) A /24 network.
- `network_id` (Number) The ID of the specific network

<a id="nestedatt--filter"></a>
### Nested Schema for `filter`

Optional:

- `network_id` (Number) The ID of the specific network, must be a positive number.
