package godaddy

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"godaddy": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	verifyEnvExists(t, "GODADDY_API_KEY")
	verifyEnvExists(t, "GODADDY_API_SECRET")
//	verifyEnvExists(t, "GODADDY_API_CUSTOMER_ID")
	verifyEnvExists(t, "GODADDY_DOMAIN")
}

func verifyEnvExists(t *testing.T, key string) {
	if v := os.Getenv(key); v == "" {
		t.Fatal(fmt.Sprintf("%s must be set for acceptance tests.", key))
	}
}
