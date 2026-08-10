package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func init() {
	resource.AddTestSweepers("xelon_object_storage_bucket", &resource.Sweeper{
		Name: "xelon_object_storage_bucket",
		F: func(region string) error {
			ctx := context.Background()
			client, err := sharedClient(region)
			if err != nil {
				return err
			}

			buckets, errf := client.ObjectStorages.AllBuckets(ctx, &xelon.ListOptions{PerPage: 100})
			for bucket := range buckets {
				if strings.HasPrefix(bucket.Name, accTestPrefix) {
					slog.Info("Deleting xelon_object_storage_bucket", "name", bucket.Name, "user_id", bucket.ObjectStorageUserID)
					_, err := client.ObjectStorages.DeleteBucket(ctx, bucket.Name, bucket.ObjectStorageUserID)
					if err != nil {
						slog.Warn("Error deleting object storage bucket during sweep", "name", bucket.Name, "user_id", bucket.ObjectStorageUserID, "error", err)
					}
				}
			}
			if err := errf(); err != nil {
				return fmt.Errorf("getting object storage bucket list: %w", err)
			}
			return nil
		},
	})
}

func TestAccResourceXelonObjectStorageBucket(t *testing.T) {
	userName := acctest.RandomWithPrefix(accTestPrefix)
	bucketName := acctest.RandomWithPrefix(accTestPrefix)
	bucketNameUpdated := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a versioned bucket and verify the default Object Lock state.
			{

				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: true,
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketName),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("user_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("versioning_enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("object_lock_enabled"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("object_lock_retention_days"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("created_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("region_replication_enabled"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("s3_endpoints"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import the bucket and verify all state can be reconstructed.
			{
				ResourceName: "xelon_object_storage_bucket.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_object_storage_bucket.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_object_storage_bucket.test")
					}
					userID := rs.Primary.Attributes["user_id"]
					name := rs.Primary.Attributes["name"]
					return fmt.Sprintf("%v/%v", userID, name), nil
				},
				ImportStateVerify: true,
			},
			// Change the bucket name and verify identity replacement keeps versioning enabled.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketNameUpdated, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: true,
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_object_storage_bucket.test", plancheck.ResourceActionReplace),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketNameUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("versioning_enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			// Disable versioning and verify the mutable setting is updated in place.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketNameUpdated, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: false,
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_object_storage_bucket.test", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketNameUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("versioning_enabled"),
						knownvalue.Bool(false),
					),
				},
			},
			// Re-enable versioning and verify the mutable setting can be restored.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketNameUpdated, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: true,
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_object_storage_bucket.test", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("versioning_enabled"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func TestAccResourceXelonObjectStorageBucket_ExternalDeletion_RemovesStateAndRecreates(t *testing.T) {
	userName := acctest.RandomWithPrefix(accTestPrefix)
	bucketName := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Delete the bucket outside Terraform after initial creation.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: false,
				}),
				Check:              testAccResourceXelonObjectStorageBucketDelete("xelon_object_storage_bucket.test"),
				ExpectNonEmptyPlan: true,
			},
			// Recreate the missing bucket after refresh removes it from state.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: false,
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketName),
					),
				},
			},
		},
	})
}

func TestAccResourceXelonObjectStorageBucket_ObjectLock(t *testing.T) {
	userName := acctest.RandomWithPrefix(accTestPrefix)
	bucketName := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a locked bucket with a 30-day retention period.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockEnabled:       new(true),
					ObjectLockRetentionDays: new(30),
					VersioningEnabled:       true,
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("object_lock_enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("object_lock_retention_days"),
						knownvalue.Int64Exact(30),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("versioning_enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			// Import the bucket and verify Object Lock state is reconstructed.
			{
				ResourceName: "xelon_object_storage_bucket.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_object_storage_bucket.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_object_storage_bucket.test")
					}
					userID := rs.Primary.Attributes["user_id"]
					name := rs.Primary.Attributes["name"]
					return fmt.Sprintf("%v/%v", userID, name), nil
				},
				ImportStateVerify: true,
			},
			// Change to a 60-day retention period and verify that replacement is required.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockEnabled:       new(true),
					ObjectLockRetentionDays: new(60),
					VersioningEnabled:       true,
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_object_storage_bucket.test", plancheck.ResourceActionReplace),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_bucket.test",
						tfjsonpath.New("object_lock_retention_days"),
						knownvalue.Int64Exact(60),
					),
				},
			},
			// Import the replaced bucket and verify retention is read back.
			{
				ResourceName: "xelon_object_storage_bucket.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_object_storage_bucket.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_object_storage_bucket.test")
					}
					userID := rs.Primary.Attributes["user_id"]
					name := rs.Primary.Attributes["name"]
					return fmt.Sprintf("%v/%v", userID, name), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceXelonObjectStorageBucket_InvalidObjectLockConfiguration(t *testing.T) {
	userName := acctest.RandomWithPrefix(accTestPrefix)
	bucketName := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Reject Object Lock when versioning is disabled.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockEnabled:       new(true),
					ObjectLockRetentionDays: new(30),
					VersioningEnabled:       false,
				}),
				ExpectError: regexp.MustCompile("versioning_enabled.*must be true"),
			},
			// Reject Object Lock without a retention period before calling the API.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockEnabled: new(true),
					VersioningEnabled: true,
				}),
				ExpectError: regexp.MustCompile("Object Lock requires retention"),
			},
			// Reject retention days when Object Lock is not enabled.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockRetentionDays: new(30),
					VersioningEnabled:       true,
				}),
				ExpectError: regexp.MustCompile("object_lock_retention_days.*can only be configured"),
			},
		},
	})
}

func TestAccResourceXelonObjectStorageBucket_ObjectLockChangeForcesReplacement(t *testing.T) {
	userName := acctest.RandomWithPrefix(accTestPrefix)
	bucketName := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create an ordinary bucket with Object Lock disabled.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					VersioningEnabled: true,
				}),
			},
			// Enable Object Lock and verify that replacement is required.
			{
				Config: testAccResourceXelonObjectStorageBucketConfig(userName, bucketName, testAccObjectStorageBucketConfigOptions{
					ObjectLockEnabled:       new(true),
					ObjectLockRetentionDays: new(30),
					VersioningEnabled:       true,
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_object_storage_bucket.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

type testAccObjectStorageBucketConfigOptions struct {
	ObjectLockEnabled       *bool
	ObjectLockRetentionDays *int
	VersioningEnabled       bool
}

func testAccResourceXelonObjectStorageBucketConfig(
	userName string,
	bucketName string,
	options testAccObjectStorageBucketConfigOptions,
) string {
	var objectLockConfig string

	if options.ObjectLockEnabled != nil {
		objectLockConfig += fmt.Sprintf(
			"  object_lock_enabled        = %t\n",
			*options.ObjectLockEnabled,
		)
	}

	if options.ObjectLockRetentionDays != nil {
		objectLockConfig += fmt.Sprintf(
			"  object_lock_retention_days = %d\n",
			*options.ObjectLockRetentionDays,
		)
	}

	return fmt.Sprintf(`
resource "xelon_object_storage_bucket" "test" {
  name                       = %q
  user_id                    = xelon_object_storage_user.test.id
%s  versioning_enabled         = %t
}

resource "xelon_object_storage_user" "test" {
  name          = %q
  region        = "zh1"
  storage_limit = 100
  tenant_id     = data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`,
		bucketName,
		objectLockConfig,
		options.VersioningEnabled,
		userName,
	)
}

func testAccResourceXelonObjectStorageBucketDelete(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		name := rs.Primary.Attributes["name"]
		userID := rs.Primary.Attributes["user_id"]

		resp, err := client.ObjectStorages.DeleteBucket(context.Background(), name, userID)
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			return err
		}

		return nil
	}
}
