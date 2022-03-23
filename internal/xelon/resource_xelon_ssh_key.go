package xelon

import (
	"context"
	"strconv"
	"strings"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceXelonSSHKey() *schema.Resource {
	return &schema.Resource{
		Description: "Xelon resource to allow you to manage SSH keys",

		CreateContext: resourceXelonSSHKeyCreate,
		ReadContext:   resourceXelonSSHKeyRead,
		UpdateContext: resourceXelonSSHKeyUpdate,
		DeleteContext: resourceXelonSSHKeyDelete,

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
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
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

	tflog.Debug(ctx, "resourceXelonSSHKeyCreate", map[string]interface{}{"payload": createRequest})
	key, _, err := client.SSHKeys.Create(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating SSH key: %s", err)
	}

	d.SetId(strconv.Itoa(key.ID))
	tflog.Info(ctx, "created SSH key", map[string]interface{}{"ssh_key_id": key.ID})

	return resourceXelonSSHKeyRead(ctx, d, meta)
}

func resourceXelonSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Invalid SSH key id: %v", err)
	}

	keys, _, err := client.SSHKeys.List(ctx)
	if err != nil {
		diag.Errorf("Error retrieving SSH key: %s", err)
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
		return diag.Errorf("Invalid SSH key id: %v", err)
	}

	tflog.Debug(ctx, "resourceXelonSSHKeyDelete", map[string]interface{}{"ssh_key_id": id})
	_, err = client.SSHKeys.Delete(ctx, id)
	if err != nil {
		return diag.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
