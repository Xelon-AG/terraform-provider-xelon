package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonDeviceBackup_Schema_ReplacementAndMutableAttributes(t *testing.T) {
	deviceBackupSchema := testDeviceBackupResourceSchema(t)

	deviceID, ok := deviceBackupSchema.Attributes["device_id"].(schema.StringAttribute)
	require.True(t, ok)
	require.Len(t, deviceID.PlanModifiers, 1)

	backupPlanID, ok := deviceBackupSchema.Attributes["backup_plan_id"].(schema.Int64Attribute)
	require.True(t, ok)
	assert.Empty(t, backupPlanID.PlanModifiers)
}

func TestResourceXelonDeviceBackup_Model_FromAPI(t *testing.T) {
	backupPlan := &xelon.BackupPlan{
		ID:   17,
		Name: "Daily Backup",
	}
	expected := deviceBackupResourceModel{
		BackupPlanID: types.Int64Value(17),
		DeviceID:     types.StringValue("device-123"),
	}

	var actual deviceBackupResourceModel
	actual.fromAPI(backupPlan, "device-123")

	assert.Equal(t, expected, actual)
}

func testDeviceBackupResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	r := NewDeviceBackupResource()
	response := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())

	return response.Schema
}
