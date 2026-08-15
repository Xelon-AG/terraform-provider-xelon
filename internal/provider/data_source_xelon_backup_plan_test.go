package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestDataSourceXelonBackupPlan_BackupPlanLookup_ExactMatch(t *testing.T) {
	backupPlans := []xelon.BackupPlan{
		{ID: 10, Name: "Daily Backup"},
		{ID: 20, Name: "Daily Backup, kept for 5 days"},
		{ID: 30, Name: "Daily Backup, kept for 5 days (legacy)"},
	}

	backupPlan, matchCount := findBackupPlanByName(backupPlans, "Daily Backup, kept for 5 days")

	require.Equal(t, 1, matchCount)
	require.NotNil(t, backupPlan)
	assert.Equal(t, 20, backupPlan.ID)
}

func TestDataSourceXelonBackupPlan_BackupPlanLookup_CaseSensitive(t *testing.T) {
	backupPlans := []xelon.BackupPlan{
		{ID: 10, Name: "Daily Backup"},
	}

	backupPlan, matchCount := findBackupPlanByName(backupPlans, "daily backup")

	assert.Nil(t, backupPlan)
	assert.Equal(t, 0, matchCount)
}

func TestDataSourceXelonBackupPlan_BackupPlanLookup_ZeroMatches(t *testing.T) {
	backupPlans := []xelon.BackupPlan{
		{ID: 10, Name: "Daily Backup"},
	}

	backupPlan, matchCount := findBackupPlanByName(backupPlans, "Weekly Backup")

	assert.Nil(t, backupPlan)
	assert.Equal(t, 0, matchCount)
}

func TestDataSourceXelonBackupPlan_BackupPlanLookup_AmbiguousMatches(t *testing.T) {
	backupPlans := []xelon.BackupPlan{
		{ID: 10, Name: "Daily Backup"},
		{ID: 20, Name: "Daily Backup"},
	}

	backupPlan, matchCount := findBackupPlanByName(backupPlans, "Daily Backup")

	require.NotNil(t, backupPlan)
	assert.Equal(t, 2, matchCount)
}
