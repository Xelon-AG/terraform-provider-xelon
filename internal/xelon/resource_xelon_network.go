package xelon

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func resourceXelonNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "The network resource allows you to manage Xelon networks.",

		CreateContext: resourceXelonNetworkCreate,
		ReadContext:   resourceXelonNetworkRead,
		UpdateContext: resourceXelonNetworkUpdate,
		DeleteContext: resourceXelonNetworkDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"cloud_id": {
				Description: "The cloud ID from your organization.",
				Required:    true,
				Type:        schema.TypeInt,
			},
			"dns_primary": {
				Description: "The primary DNS server address.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"dns_secondary": {
				Description: "The secondary DNS server address.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"gateway": {
				Description: "The default gateway IP address.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "The name of the network.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"netmask": {
				Description: "The netmask of the network.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"network": {
				Description: "A /24 network.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"type": {
				Description:  "The network type. Must be one of `WAN` or `LAN`.",
				Required:     true,
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{"LAN"}, false),
			},
		},
	}
}

func resourceXelonNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	tenant, err := fetchTenant(ctx, client)
	if err != nil {
		return diag.FromErr(err)
	}

	createRequest := &xelon.NetworkLANCreateRequest{
		CloudID:      d.Get("cloud_id").(int),
		DisplayName:  d.Get("name").(string),
		DNSPrimary:   d.Get("dns_primary").(string),
		DNSSecondary: d.Get("dns_secondary").(string),
		Gateway:      d.Get("gateway").(string),
		Network:      d.Get("network").(string),
	}

	tflog.Debug(ctx, "resourceXelonNetworkCreate", map[string]interface{}{
		"payload": createRequest,
	})
	_, _, err = client.Networks.CreateLAN(ctx, tenant.TenantID, createRequest)
	if err != nil {
		return diag.Errorf("creating network, %s", err)
	}

	// find networks because API doesn't return network ID.
	var network *xelon.Network
	networks, _, err := client.Networks.List(ctx, tenant.TenantID)
	if err != nil {
		return diag.Errorf("listing networks, %s", err)
	}
	for _, n := range networks {
		n := n
		if n.Name == d.Get("name").(string) && n.Network == d.Get("network").(string) {
			network = &n
			break
		}
	}
	if network == nil {
		return diag.Errorf("not found created network")
	}

	d.SetId(strconv.Itoa(network.ID))
	tflog.Info(ctx, "created network", map[string]interface{}{
		"network_id": network.ID,
	})

	return resourceXelonNetworkRead(ctx, d, meta)
}

func resourceXelonNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	tenant, err := fetchTenant(ctx, client)
	if err != nil {
		return diag.FromErr(err)
	}

	networkID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("invalid network id: %v", err)
	}

	n, resp, err := client.Networks.Get(ctx, tenant.TenantID, networkID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			tflog.Warn(ctx, "Network not found, removing from state", map[string]interface{}{
				"network_id": d.Id(),
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("getting network: %s", err)
	}

	network := n.Details
	_ = d.Set("cloud_id", network.HVSystemID)
	_ = d.Set("dns_primary", network.DNSPrimary)
	_ = d.Set("dns_secondary", network.DNSSecondary)
	_ = d.Set("gateway", network.DefaultGateway)
	_ = d.Set("name", network.Name)
	_ = d.Set("netmask", network.Netmask)
	_ = d.Set("network", network.Network)
	_ = d.Set("type", network.Type)

	return nil
}

func resourceXelonNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	networkID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("invalid network id: %v", err)
	}

	updateRequest := &xelon.NetworkUpdateRequest{
		NetworkDetails: &xelon.NetworkDetails{
			DefaultGateway: d.Get("gateway").(string),
			DNSPrimary:     d.Get("dns_primary").(string),
			DNSSecondary:   d.Get("dns_secondary").(string),
			HVSystemID:     d.Get("cloud_id").(int),
			Name:           d.Get("name").(string),
			Network:        d.Get("network").(string),
			Type:           d.Get("type").(string),
		},
	}

	tflog.Debug(ctx, "resourceXelonNetworkUpdate", map[string]interface{}{
		"payload": updateRequest,
	})
	_, _, err = client.Networks.Update(ctx, networkID, updateRequest)
	if err != nil {
		return diag.Errorf("updating network: %s", err)
	}

	tflog.Info(ctx, "updated network", map[string]interface{}{
		"network_id": d.Id(),
	})

	return resourceXelonNetworkRead(ctx, d, meta)
}

func resourceXelonNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	networkID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("invalid network id: %v", err)
	}

	tflog.Debug(ctx, "resourceXelonNetworkDelete", map[string]interface{}{
		"network_id": d.Id(),
	})
	_, err = client.Networks.Delete(ctx, networkID)
	if err != nil {
		return diag.Errorf("deleting network: %s", err)
	}

	d.SetId("")
	return nil
}
