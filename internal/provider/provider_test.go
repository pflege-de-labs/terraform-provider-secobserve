package provider_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/provider"
)

// testAccProtoV6ProviderFactories wires the in-process provider into the
// acceptance test framework. Referenced by every acceptance test in the repo.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"secobserve": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck fails early with an actionable message instead of letting
// the provider's Configure emit a less specific error per test case.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, envVar := range []string{"SECOBSERVE_BASE_URL", "SECOBSERVE_API_TOKEN"} {
		if os.Getenv(envVar) == "" {
			t.Fatalf("%s must be set for acceptance tests; run: make up && eval \"$(./test/bootstrap.sh)\"", envVar)
		}
	}
}

// acceptanceName builds a name unlikely to collide with anything already on
// the instance. Product and product group names share one unique constraint,
// so a collision fails the whole test rather than just one step.
func acceptanceName(prefix string) string {
	return fmt.Sprintf("tfacc-%s-%d", prefix, time.Now().UnixNano())
}
