package device

// import (
// 	"context"
// 	"fmt"
// 	"time"
//
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
//
// 	"github.com/Xelon-AG/xelon-sdk-go/xelon"
// )
//
// func WaitPowerStateOn(ctx context.Context, client *xelon.Client, tenantID, localVMID string) error {
// 	stateConf := &retry.StateChangeConf{
// 		Pending: []string{devicePowerStateOff},
// 		Target:  []string{devicePowerStateOn},
// 		Timeout: 10 * time.Minute,
// 		Delay:   5 * time.Second,
// 		Refresh: statusPowerState(ctx, client, tenantID, localVMID),
// 	}
//
// 	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
// 		return fmt.Errorf("waiting for device (%s) to become PoweredOn: %w", localVMID, err)
// 	}
//
// 	return nil
// }
//
// func WaitPowerStateOff(ctx context.Context, client *xelon.Client, tenantID, localVMID string) error {
// 	stateConf := &retry.StateChangeConf{
// 		Pending: []string{devicePowerStateOn},
// 		Target:  []string{devicePowerStateOff},
// 		Timeout: 10 * time.Minute,
// 		Delay:   5 * time.Second,
// 		Refresh: statusPowerState(ctx, client, tenantID, localVMID),
// 	}
//
// 	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
// 		return fmt.Errorf("waiting for device (%s) to become PoweredOff: %w", localVMID, err)
// 	}
//
// 	return nil
// }
//
// func WaitVMWareToolsStatusRunning(ctx context.Context, client *xelon.Client, tenantID, localVMID string) error {
// 	stateConf := &retry.StateChangeConf{
// 		Pending: []string{deviceVMWareToolsStatusNotRunning},
// 		Target:  []string{deviceVMWareToolsStatusRunning},
// 		Timeout: 10 * time.Minute,
// 		Delay:   5 * time.Second,
// 		Refresh: statusVMWareToolsStatus(ctx, client, tenantID, localVMID),
// 	}
//
// 	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
// 		return fmt.Errorf("waiting for device (%s) to become VMWare Tools running : %w", localVMID, err)
// 	}
//
// 	return nil
// }
