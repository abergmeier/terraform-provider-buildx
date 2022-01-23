package instancetest

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccInstance_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance(rName),
				Check:  resource.ComposeTestCheckFunc(),
			},
			{
				Config: testAccInstanceBootstrapped(rName),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAccInstance(rName string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver {
    name = "docker-container"
  }
}
`, rName)
}

func testAccInstanceBootstrapped(rName string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver {
    name = "docker-container"
  }
  bootstrap = true
}
`, rName)
}
