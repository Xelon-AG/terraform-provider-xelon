package xelon

// import (
// 	"context"
// 	"fmt"
// 	"strconv"
//
// 	"github.com/Xelon-AG/xelon-sdk-go/xelon"
// 	"github.com/hashicorp/terraform-plugin-log/tflog"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
//
// 	"github.com/Xelon-AG/terraform-provider-xelon/internal/xelon/storage"
// )
//
// func resourceXelonPersistentStorage() *schema.Resource {
// 	return &schema.Resource{
// 		Description: "The persistent storage resource allows you to manage Xelon Persistent Storages.",
//
// 		CreateContext: resourceXelonPersistentStorageCreate,
// 		ReadContext:   resourceXelonPersistentStorageRead,
// 		UpdateContext: resourceXelonPersistentStorageUpdate,
// 		DeleteContext: resourceXelonPersistentStorageDelete,
//
// 		SchemaVersion: 0,
//
// 		Schema: map[string]*schema.Schema{
// 			"cloud_id": {
// 				Description: "The ID of the organization cloud.",
// 				ForceNew:    true,
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"formatted": {
// 				Description: "True, if the persistent storage is formatted.",
// 				Computed:    true,
// 				Type:        schema.TypeBool,
// 			},
// 			"name": {
// 				Description: "The name of the persistent storage.",
// 				ForceNew:    true,
// 				Required:    true,
// 				Type:        schema.TypeString,
// 			},
// 			"size": {
// 				Description: "The size of the persistent storage in GB. If updated, can only be expanded.",
// 				Required:    true,
// 				Type:        schema.TypeInt,
// 			},
// 			"uuid": {
// 				Description: "The UUID of the persistent storage.",
// 				Computed:    true,
// 				Type:        schema.TypeString,
// 			},
// 		},
//
// 		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
// 			oldSize, newSize := diff.GetChange("size")
// 			if newSize.(int) < oldSize.(int) {
// 				return fmt.Errorf("persistent storages 'size' can only be expanded")
// 			}
// 			return nil
// 		},
// 	}
// }
//
// func resourceXelonPersistentStorageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	cloudID := strconv.Itoa(d.Get("cloud_id").(int))
// 	createRequest := &xelon.PersistentStorageCreateRequest{
// 		PersistentStorage: &xelon.PersistentStorage{
// 			Name: d.Get("name").(string),
// 			Type: 2,
// 		},
// 		CloudID: cloudID,
// 		Size:    d.Get("size").(int),
// 	}
//
// 	tflog.Debug(ctx, "resourceXelonPersistentStorageCreate", map[string]interface{}{
// 		"payload": createRequest,
// 	})
// 	apiResponse, _, err := client.PersistentStorages.Create(ctx, tenant.TenantID, createRequest)
// 	if err != nil {
// 		return diag.Errorf("creating persistent storage: %s", err)
// 	}
//
// 	localID := apiResponse.PersistentStorage.LocalID
// 	d.SetId(localID)
//
// 	tflog.Info(ctx, "created persistent storage", map[string]interface{}{
// 		"persistent_storage_id": localID,
// 	})
//
// 	err = storage.WaitStorageStateCreated(ctx, client, tenant.TenantID, localID)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	return resourceXelonPersistentStorageRead(ctx, d, meta)
// }
//
// func resourceXelonPersistentStorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	persistentStorage, resp, err := client.PersistentStorages.Get(ctx, tenant.TenantID, d.Id())
// 	if err != nil {
// 		if resp != nil && resp.StatusCode == 404 {
// 			tflog.Warn(ctx, "Persistent storage not found, removing from state", map[string]interface{}{
// 				"persistent_storage_id": d.Id(),
// 			})
// 			d.SetId("")
// 			return nil
// 		}
// 		return diag.Errorf("getting persistent storage: %s", err)
// 	}
//
// 	_ = d.Set("formatted", persistentStorage.Formatted == 1)
// 	_ = d.Set("name", persistentStorage.Name)
// 	_ = d.Set("size", persistentStorage.Capacity)
// 	_ = d.Set("uuid", persistentStorage.UUID)
//
// 	return nil
// }
//
// func resourceXelonPersistentStorageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	if d.HasChange("size") {
// 		extendRequest := &xelon.PersistentStorageExtendRequest{
// 			Size: d.Get("size").(int),
// 		}
//
// 		tflog.Debug(ctx, "resourceXelonPersistentStorageUpdate", map[string]interface{}{
// 			"payload": extendRequest,
// 		})
// 		_, _, err := client.PersistentStorages.Extend(ctx, d.Id(), extendRequest)
// 		if err != nil {
// 			return diag.Errorf("extending persistent storage: %s", err)
// 		}
//
// 		tflog.Info(ctx, "extended persistent storage", map[string]interface{}{
// 			"persistent_storage_id": d.Id(),
// 		})
//
// 		tenant, err := fetchTenant(ctx, client)
// 		if err != nil {
// 			return diag.FromErr(err)
// 		}
// 		err = storage.WaitStorageStateCreated(ctx, client, tenant.TenantID, d.Id())
// 		if err != nil {
// 			return diag.FromErr(err)
// 		}
// 	}
//
// 	return resourceXelonPersistentStorageRead(ctx, d, meta)
// }
//
// func resourceXelonPersistentStorageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	client := meta.(*xelon.Client)
//
// 	tenant, err := fetchTenant(ctx, client)
// 	if err != nil {
// 		return diag.FromErr(err)
// 	}
//
// 	tflog.Debug(ctx, "resourceXelonPersistentStorageDelete", map[string]interface{}{
// 		"persistent_storage_id": d.Id(),
// 	})
// 	_, err = client.PersistentStorages.Delete(ctx, tenant.TenantID, d.Id())
// 	if err != nil {
// 		return diag.Errorf("deleting persistent storage: %s", err)
// 	}
//
// 	d.SetId("")
// 	return nil
// }
