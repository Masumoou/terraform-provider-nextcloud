package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

// configureClient safely type-asserts the shared *client.Client out of
// ProviderData, adding a diagnostic and returning nil on mismatch instead of
// panicking. Called from every resource/data source's Configure method.
func configureClient(providerData interface{}, diags *diag.Diagnostics) *client.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. This is a provider bug — please report it.", providerData),
		)
		return nil
	}
	return c
}

// diagAppender is a tiny wrapper so resource-specific helper methods (like
// OnlyOfficeResource.applyAndValidate) can add diagnostics without needing
// the full *resource.CreateResponse/UpdateResponse type.
type diagAppender struct {
	diags *diag.Diagnostics
}

func (d *diagAppender) AddError(summary, detail string)   { d.diags.AddError(summary, detail) }
func (d *diagAppender) AddWarning(summary, detail string) { d.diags.AddWarning(summary, detail) }

// joinStrings is a tiny, dependency-free strings.Join to keep imports
// minimal in resource files that only need this once.
func joinStrings(in []string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// stringListValue converts a []string into a types.List of strings,
// swallowing conversion diagnostics since a []string can never fail this
// conversion.
func stringListValue(in []string) types.List {
	elems := make([]types.String, 0, len(in))
	for _, s := range in {
		elems = append(elems, types.StringValue(s))
	}
	l, _ := types.ListValueFrom(context.Background(), types.StringType, elems)
	return l
}
