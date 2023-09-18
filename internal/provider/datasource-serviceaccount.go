// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	keyhub "github.com/topicuskeyhub/sdk-go"
	keyhubmodels "github.com/topicuskeyhub/sdk-go/models"
	keyhubreq "github.com/topicuskeyhub/sdk-go/serviceaccount"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &serviceaccountDataSource{}
	_ datasource.DataSourceWithConfigure = &serviceaccountDataSource{}
)

func NewServiceaccountDataSource() datasource.DataSource {
	return &serviceaccountDataSource{}
}

type serviceaccountDataSource struct {
	client *keyhub.KeyHubClient
}

func (d *serviceaccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serviceaccount"
}

func (d *serviceaccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: dataSourceSchemaAttrsServiceaccountServiceAccount(true),
	}
}

func (d *serviceaccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*keyhub.KeyHubClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *keyhub.KeyHubClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *serviceaccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceaccountServiceAccountDataDS
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading serviceaccount from Topicus KeyHub by UUID")
	listValue, _ := data.Additional.ToListValue(ctx)
	additional, _ := tfToSlice(listValue, func(val attr.Value, diags *diag.Diagnostics) string {
		return val.(basetypes.StringValue).ValueString()
	})
	uuid := data.UUID.ValueString()

	wrapper, err := d.client.Serviceaccount().Get(ctx, &keyhubreq.ServiceaccountRequestBuilderGetRequestConfiguration{
		QueryParameters: &keyhubreq.ServiceaccountRequestBuilderGetQueryParameters{
			Uuid:       []string{uuid},
			Additional: additional,
		},
	})

	tkh, diags := findFirst[keyhubmodels.ServiceaccountServiceAccountable](ctx, wrapper, "serviceaccount", &uuid, err)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tf, diags := tkhToTFObjectDSServiceaccountServiceAccount(true, tkh)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fillDataStructFromTFObjectDSServiceaccountServiceAccount(&data, tf)
	data.Additional = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
