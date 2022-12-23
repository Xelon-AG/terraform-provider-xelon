
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
