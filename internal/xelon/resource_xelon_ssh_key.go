package xelon

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func resourceXelonSSHKey() *schema.Resource {
	return &schema.Resource{
		Description: "Xelon resource to allow you to manage SSH keys",

		CreateContext: resourceXelonSSHKeyCreate,
		ReadContext:   resourceXelonSSHKeyRead,
		UpdateContext: resourceXelonSSHKeyUpdate,
		DeleteContext: resourceXelonSSHKeyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fingerprint of the SSH key",
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the SSH key",
				ValidateFunc: validation.NoZeroValues,
			},

			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The text of the public key",
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return strings.TrimSpace(oldValue) == strings.TrimSpace(newValue)
				},
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceXelonSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	createRequest := &xelon.SSHKeyCreateRequest{
		SSHKey: &xelon.SSHKey{
			Name:      d.Get("name").(string),
			PublicKey: d.Get("public_key").(string),
		},
	}

	tflog.Debug(ctx, "Creating Xelon SSH Key", map[string]interface{}{
		"configuration": createRequest,
	})

	key, _, err := client.SSHKeys.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating Xelon SSH Key: %w", err))
	}

	d.SetId(strconv.Itoa(key.ID))

	tflog.Info(ctx, "Created Xelon SSH Key", map[string]interface{}{
		"id":          key.ID,
		"fingerprint": key.Fingerprint,
	})

	return resourceXelonSSHKeyRead(ctx, d, meta)
}

func resourceXelonSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid Xelon SSH Key ID: %w", err))
	}

	keys, _, err := client.SSHKeys.List(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("retrieving Xelon SSH Keys: %w", err))
	}
	for _, key := range keys {
		// will be refactored later when get method for single key is available
		if key.ID == id {
			_ = d.Set("fingerprint", key.Fingerprint)
			_ = d.Set("name", key.Name)
			return nil
		}
	}

	return nil
}

func resourceXelonSSHKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// workaround because of missing update method for single key
	resourceXelonSSHKeyDelete(ctx, d, meta)
	resourceXelonSSHKeyCreate(ctx, d, meta)

	return nil
}

func resourceXelonSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid Xelon SSH Key ID: %w", err))
	}

	tflog.Debug(ctx, "Deleting Xelon SSH Key", map[string]interface{}{
		"id": id,
	})

	_, err = client.SSHKeys.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("deleting Xelon SSH Key: %w", err))
	}

	d.SetId("")

	return nil
}
