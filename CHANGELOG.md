
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
