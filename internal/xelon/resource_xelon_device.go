package xelon

// import (
// 	"context"
// 	"fmt"
//
// 	"github.com/Xelon-AG/xelon-sdk-go/xelon"
// 	"github.com/hashicorp/terraform-plugin-log/tflog"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
//
// 	"github.com/Xelon-AG/terraform-provider-xelon/internal/xelon/device"
// )
//
// func resourceXelonDevice() *schema.Resource {
// 	return &schema.Resource{
// 		Description: "The device resource allows you to manage Xelon devices.",
//
// 		CreateContext: resourceXelonDeviceCreate,
// 		ReadContext:   resourceXelonDeviceRead,
// 		UpdateContext: resourceXelonDeviceUpdate,
// 		DeleteContext: resourceXelonDeviceDelete,
//
// 		SchemaVersion: 0,
//
// 		Schema: map[string]*schema.Schema{
// 			"cloud_id": {
// 				Description: "The cloud ID from your organization.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"cpu_core_count": {
// 				Description: "The number of CPU cores for a device",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"disk_size": {
// 				Description: "Size of a disk in gigabytes.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"display_name": {
// 				Description: "Display name of a device.",
// 				Required:    true,
// 				Type:        schema.TypeString,
// 			},
// 			"hostname": {
// 				Description: "Hostname of a device.",
// 				Required:    true,
// 				Type:        schema.TypeString,
// 			},
// 			"memory": {
// 				Description: "Amount of RAM in gigabytes.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"network": {
// 				Description: "Device network interface configuration.",
// 				MaxItems:    1,
// 				Required:    true,
// 				Type:        schema.TypeList,
// 				Elem: &schema.Resource{
// 					Schema: map[string]*schema.Schema{
// 						"id": {
// 							Description: "Network ID available for your organization.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 						"ipv4_address_id": {
// 							Description: "IPv4 address ID for a device.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 						"nic_controller_key": {
// 							Description: "Network interface card (NIC) controller key.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 						"nic_key": {
// 							Description: "Network interface card (NIC) key.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 						"nic_number": {
// 							Description: "Network interface card (NIC) number.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 						"nic_unit": {
// 							Description: "Network interface card (NIC) unit.",
// 							Required:    true,
// 							Type:        schema.TypeInt,
// 						},
// 					},
// 				},
// 			},
// 			"password": {
// 				Description: "Password of a device.",
// 				Required:    true,
// 				Sensitive:   true,
// 				Type:        schema.TypeString,
// 			},
// 			"swap_disk_size": {
// 				Description: "Size of a SWAP disk in gigabytes.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"template_id": {
// 				Description: "Template ID of the selected OS.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 		},
// 	}
// }
//
// func resourceXelonDeviceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	createRequest := &xelon.DeviceCreateRequest{
// 		CloudID:              d.Get("cloud_id").(int),
// 		CPUCores:             d.Get("cpu_core_count").(int),
// 		DiskSize:             d.Get("disk_size").(int),
// 		DisplayName:          d.Get("display_name").(string),
// 		Hostname:             d.Get("hostname").(string),
// 		Memory:               d.Get("memory").(int),
// 		Password:             d.Get("password").(string),
// 		PasswordConfirmation: d.Get("password").(string),
// 		SwapDiskSize:         d.Get("swap_disk_size").(int),
// 		TemplateID:           d.Get("template_id").(int),
// 		TenantID:             tenant.TenantID,
// 	}
// 	if n, ok := d.GetOk("network"); ok {
// 		network := device.ExpandNetwork(n.([]interface{}))
//
// 		createRequest.IPAddressID = network.IPAddressID
// 		createRequest.NetworkID = network.NetworkID
// 		createRequest.NICControllerKey = network.NICControllerKey
// 		createRequest.NICKey = network.NICKey
// 		createRequest.NICNumber = network.NICNumber
// 		createRequest.NICUnit = network.NICUnit
// 	}
//
// 	tflog.Debug(ctx, "resourceXelonDeviceCreate", map[string]interface{}{
// 		"payload": createRequest,
// 	})
// 	deviceCreateResponse, _, err := client.Devices.Create(ctx, createRequest)
// 	if err != nil {
// 		return diag.Errorf("creating device: %s", err)
// 	}
//
// 	localVMID := deviceCreateResponse.LocalVMDetails.LocalVMID
// 	d.SetId(localVMID)
//
// 	tflog.Info(ctx, "created device", map[string]interface{}{
// 		"device_id": localVMID,
// 	})
//
// 	err = device.WaitPowerStateOn(ctx, client, tenant.TenantID, localVMID)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	err = device.WaitVMWareToolsStatusRunning(ctx, client, tenant.TenantID, localVMID)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	return resourceXelonDeviceRead(ctx, d, meta)
// }
//
// func resourceXelonDeviceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	_, resp, err := client.Devices.Get(ctx, tenant.TenantID, d.Id())
// 	if err != nil {
// 		if resp != nil && resp.StatusCode == 404 {
// 			tflog.Warn(ctx, "Device not found, removing from state", map[string]interface{}{
// 				"device_id": d.Id(),
// 			})
// 			d.SetId("")
// 			return nil
// 		}
// 		return diag.Errorf("getting device: %s", err)
// 	}
//
// 	return nil
// }
//
// func resourceXelonDeviceUpdate(ctx context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
// 	tflog.Warn(ctx, "resourceXelonDeviceUpdate is not implemented")
// 	return nil
// }
//
// func resourceXelonDeviceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	_, err = client.Devices.Stop(ctx, d.Id())
// 	if err != nil {
// 		return diag.Errorf("stopping device: %s", err)
// 	}
//
// 	err = device.WaitPowerStateOff(ctx, client, tenant.TenantID, d.Id())
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	tflog.Debug(ctx, "resourceXelonDeviceDelete", map[string]interface{}{
// 		"device_id": d.Id(),
// 	})
// 	_, err = client.Devices.Delete(ctx, d.Id())
// 	if err != nil {
// 		return diag.Errorf("deleting device: %s", err)
// 	}
//
// 	d.SetId("")
// 	return nil
// }
//
// func fetchTenant(ctx context.Context, client *xelon.Client) (*xelon.Tenant, error) {
// 	tenant, _, err := client.Tenants.GetCurrent(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("getting tenant information: %w", err)
// 	}
//
// 	tflog.Debug(ctx, "retrieved tenant information", map[string]interface{}{
// 		"tenant_id": tenant.TenantID,
// 	})
//
// 	return tenant, nil
// }
