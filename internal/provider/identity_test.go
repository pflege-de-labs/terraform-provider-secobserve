package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/provider"
)

// identityResources are the resource types that declare a resource identity.
var identityResources = []string{
	"secobserve_product", "secobserve_product_group", "secobserve_user",
	"secobserve_authorization_group", "secobserve_general_rule", "secobserve_license_group",
	"secobserve_license_policy", "secobserve_branch", "secobserve_service",
	"secobserve_api_configuration", "secobserve_product_rule",
}

// State without identity (Terraform 1.11, or written by provider 0.2.0) must
// still drop a resource deleted outside Terraform instead of failing with
// "Missing Resource Identity After Read".
func TestReadRemovesDeletedResourceWithoutIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/me/":
			_, _ = w.Write([]byte(`{"id":1,"username":"admin","is_superuser":true}`))
		case "/api/status/version/":
			_, _ = w.Write([]byte(`{"version":"1.60.0"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		}
	}))
	defer server.Close()
	t.Setenv("SECOBSERVE_BASE_URL", server.URL)
	t.Setenv("SECOBSERVE_API_TOKEN", "t")

	ctx := context.Background()
	srv, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	providerType := schemas.Provider.ValueType()
	providerConfig, _ := tfprotov6.NewDynamicValue(providerType, nullObject(providerType.(tftypes.Object), nil))
	configured, err := srv.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: &providerConfig})
	if err != nil || len(configured.Diagnostics) > 0 {
		t.Fatalf("ConfigureProvider: %v %+v", err, configured.Diagnostics)
	}

	for _, typeName := range identityResources {
		t.Run(typeName, func(t *testing.T) {
			objectType := schemas.ResourceSchemas[typeName].ValueType().(tftypes.Object)
			known := map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Number, 1)}
			if _, ok := objectType.AttributeTypes["product"]; ok {
				known["product"] = tftypes.NewValue(tftypes.Number, 1)
			}
			state, _ := tfprotov6.NewDynamicValue(objectType, nullObject(objectType, known))

			resp, err := srv.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: &state})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.Diagnostics {
				t.Errorf("%s: %s", d.Summary, d.Detail)
			}
		})
	}
}

// nullObject builds an object whose attributes are null except for known.
func nullObject(objectType tftypes.Object, known map[string]tftypes.Value) tftypes.Value {
	values := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	for name, value := range known {
		values[name] = value
	}
	return tftypes.NewValue(objectType, values)
}
