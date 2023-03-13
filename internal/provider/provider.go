package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	defaultBaseURL = "https://hq.xelon.ch/api/service/"
)

var _ provider.Provider = (*xelonProvider)(nil)

// xelonProvider defines the provider implementation.
type xelonProvider struct {
	// version is set to
	//  - the provider version on release
	//  - "dev" when the provider is built and ran locally
	//  - "testacc" when running acceptance tests
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &xelonProvider{
			version: version,
		}
	}
}

func (p *xelonProvider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "xelon"
	response.Version = p.version
}

func (p *xelonProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional: true,
				Description: fmt.Sprintf("The base URL endpoint for Xelon HQ. Default is `%s`. "+
					"Alternatively, can be configured using the `XELON_BASE_URL` environment variable.", defaultBaseURL),
			},

			"client_id": schema.StringAttribute{
				Optional: true,
				Description: "The client ID for IP ranges. Alternatively, can be configured " +
					"using the `XELON_CLIENT_ID` environment variable.",
			},

			"token": schema.StringAttribute{
				Optional: true,
				Description: "The Xelon access token. Alternatively, can be configured " +
					"using the `XELON_TOKEN` environment variable.",
			},
		},
	}
}

type providerModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	ClientID types.String `tfsdk:"client_id"`
	Token    types.String `tfsdk:"token"`
}

func (p *xelonProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var config providerModel

	diags := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// fallback to env if unset
	if config.BaseURL.IsNull() {
		if baseURLFromEnv, ok := os.LookupEnv("XELON_BASE_URL"); ok {
			config.BaseURL = types.StringValue(baseURLFromEnv)
		} else {
			config.BaseURL = types.StringValue(defaultBaseURL)
		}
	}
	if config.ClientID.IsNull() {
		config.ClientID = types.StringValue(os.Getenv("XELON_CLIENT_ID"))
	}
	if config.Token.IsNull() {
		config.Token = types.StringValue(os.Getenv("XELON_TOKEN"))
	}

	// required if still unset
	if config.Token.ValueString() == "" {
		response.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Invalid provider config",
			"token must be set.",
		)
		return
	}

	// build xelon sdk client
	opts := []xelon.ClientOption{xelon.WithUserAgent(p.userAgent())}
	opts = append(opts, xelon.WithBaseURL(config.BaseURL.ValueString()))
	if config.ClientID.ValueString() != "" {
		opts = append(opts, xelon.WithClientID(config.ClientID.ValueString()))
	}
	client := xelon.NewClient(config.Token.ValueString(), opts...)

	tflog.Info(ctx, "Xelon SDK client configured", map[string]interface{}{
		"base_url":          config.BaseURL.ValueString(),
		"client_id":         config.ClientID.ValueString(),
		"terraform_version": request.TerraformVersion,
	})

	response.DataSourceData = client
	response.ResourceData = client
}

func (p *xelonProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNetworkDataSource,
	}
}

func (p *xelonProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *xelonProvider) userAgent() string {
	name := "terraform-provider-xelon"
	comment := "https://registry.terraform.io/providers/Xelon-AG/xelon"

	return fmt.Sprintf("%s/%s (+%s)", name, p.version, comment)
}
