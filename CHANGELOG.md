
<a name="v1.0.0-rc9"></a>
## v1.0.0-rc9 (2025-03-19)
### Bug Fixes
* **resource/xelon_load_balancer**: change type for assigned devices
### Documentation
* **resource/xelon_load_balancer_forwarding_rule**: generate documentation with example
### Features
* **resource/xelon_load_balancer**: allow to assign devices to load balancer
* **resource/xelon_load_balancer_forwarding_rule**: add new resource to create forwarding rules

<a name="v1.0.0-rc8"></a>
## v1.0.0-rc8 (2025-03-14)
### Bug Fixes
* **resource/xelon_load_balancer**: make network_id mandatory
### Features
* **resource/xelon_load_balancer**: use v2 sdk endpoints

<a name="v1.0.0-rc7"></a>
## v1.0.0-rc7 (2025-02-13)
### Bug Fixes
* **resource/xelon_network**: make required attributes configurable for LAN and WAN type
### Features
* **resource/xelon_device**: update required and optional attributes for network config
* **resource/xelon_network**: use api v2 for create/update WAN networks

<a name="v1.0.0-rc6"></a>
## v1.0.0-rc6 (2025-02-11)
### Bug Fixes
* **datasource/xelon_tenant**: do not search by name for current tenant
* **resource/xelon_device**: re-create device if template is changed
### Documentation
* fix formatting in template examples
### Features
* **datasource/xelon_tenant**: use v2 endpoints for clouds

<a name="v1.0.0-rc5"></a>
## v1.0.0-rc5 (2025-02-10)
### Bug Fixes
* **datasource/xelon_network**: fetch missing properties after listing network
### Documentation
* **datasource/xelon_network**: add examples for network filtering
### Features
* **datasource/xelon_cloud**: use v2 endpoints for clouds

<a name="v1.0.0-rc4"></a>
## v1.0.0-rc4 (2025-02-09)
### Features
* **resource/xelon_network**: use v2 sdk to create LAN networks
### Maintaining
* **deps**: upgrade hashicorp dependencies to latest versions

<a name="v1.0.0-rc3"></a>
## v1.0.0-rc3 (2025-02-07)
### Bug Fixes
* **resource/xelon_device**: relax check for power state and fresh created devices
### Features
* **resource/xelon_device**: add optional fields for device

<a name="v1.0.0-rc2"></a>
## v1.0.0-rc2 (2025-02-05)
### Bug Fixes
* **resource/xelon_device**: improve check for powered and ready state
* **resource/xelon_device**: make password sensitive
### Documentation
* **resource/xelon_device**: update documentation for network block

<a name="v1.0.0-rc1"></a>
## v1.0.0-rc1 (2025-02-05)
### Features
* **resource/xelon_device**: use xelon-sdk-go v2 endpoint
### Maintaining
* update goreleaser deprecated properties

<a name="v1.0.0-rc0"></a>
## v1.0.0-rc0 (2025-02-03)
### Bug Fixes
* resolve panic issue with SDKv2 resource import
* **lint**: replace depcreated exportloopref linter
* **linter**: resolve golangci-lint issues
* **resource/xelon_network**: use correct attributes for cloud
### Documentation
* re-generate docs after tfplugindocs update
### Features
* **datasource/xelon_network**: use xelon-sdk-go v2 endpoint
* **resource/xelon_ssh_key**: add acceptance tests
* **resource/xelon_ssh_key**: migrate ssh key resource to framework
### Maintaining
* disable SDKv2 sweeper
* configure acceptance test matrix for terraform
* **deps**: update terraform-plugin-framework-validators to v0.13.0
* **deps**: update xelon-sdk-go to v0.14.1
* **deps**: upgrade dependencies to latest stable versions
* **gh-actions**: upgrade github actions to latest stable versions
* **tools**: upgrade golangci-lint to v1.63.4
* **tools**: upgrade terraform-plugin-docs to v0.20.1
* **tools**: upgrade tools dependencies to latest stable versions

<a name="v0.7.0"></a>
## v0.7.0 (2023-03-13)
### Code Refactoring
* standardize logging by resource methods
### Documentation
* describe optional envvars for provider configuration
### Features
* **datasource/xelon_network**: add new data source for networks
* **provider**: make framework usable
### Maintaining
* update template for pull requests
* upgrade protocol version from 5 to 6
* add template for GitHub pull requests
* add template for GitHub issues
* log info when configuring SDK client
* replace deprecated GoReleaser options

<a name="v0.6.2"></a>
## v0.6.2 (2023-02-24)
### Maintaining
* **deps**: bump golang.org/x/net from 0.6.0 to 0.7.0
* **deps**: upgrade dependencies
* **tools**: upgrade tools dependencies

<a name="v0.6.1"></a>
## v0.6.1 (2023-01-01)
### Documentation
* generate examples for all resources

<a name="v0.6.0"></a>
## v0.6.0 (2022-12-31)
### Features
* **datasource/xelon_cloud**: add new data source for clouds
### Maintaining
* upgrade xelon-sdk-go to v0.12.0

<a name="v0.5.1"></a>
## v0.5.1 (2022-12-30)
### Maintaining
* run sweepers in GitHub actions after acceptance tests
* add sweepers to cleanup leftover infrastructure

<a name="v0.5.0"></a>
## v0.5.0 (2022-12-30)
### Features
* **resource/xelon_network**: add new resource for networks
### Maintaining
* upgrade xelon-sdk-go to v0.11.0
* execute acceptance tests by pull request checks
* **resource/xelon_network**: add acceptance tests
* **resource/xelon_ssh_key**: verify if key exists when executing acceptance tests
* **resource/xelon_ssh_key**: enable acceptance tests

<a name="v0.4.0"></a>
## v0.4.0 (2022-12-27)
### Features
* **resource/xelon_persistent_storage**: implement extending storage by update method
* **resource/xelon_persistent_storage**: add new resource for persistent storages
### Maintaining
* upgrade xelon-sdk-go to v0.10.1

<a name="v0.3.0"></a>
## v0.3.0 (2022-12-26)
### Features
* add client_id support for config
* **datasource/xelon_tenant**: add new data source for organizations

<a name="v0.2.0"></a>
## v0.2.0 (2022-12-23)
### Documentation
* generate provider documentation with tfplugindocs
* fix provider links for registry
### Features
* **resource/xelon_device**: add new resource for devices

<a name="v0.1.0"></a>
## v0.1.0 (2022-12-02)
### Features
* **resource/xelon_ssh_key**: add new resource for ssh keys
### Maintaining
* add GitHub release workflow
* inject provider version via ldflags
* use goreleaser to build executables
* upgrade xelon-sdk-go to v0.7.0
* use tools-as-dependency pattern for dev dependencies
* replace logging with tflog package
* use GitHub actions for running unit tests
