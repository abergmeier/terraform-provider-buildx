package instancetest

import (
	"fmt"
	"testing"

	"github.com/abergmeier/terraform-provider-buildx/internal/testproviderfactory"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccInstance_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testproviderfactory.SingleFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("buildx_instance.foo", "generated_name"),
				),
			},
			{
				Config: testAccInstanceBootstrapped(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("buildx_instance.foo", "generated_name"),
				),
			},
			{
				Config: testAccInstanceGenerated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("buildx_instance.foo", "name"),
					testAccNameGenerated("buildx_instance.foo", rName),
				),
			},
		},
	})
}

func testAccNameGenerated(resourceName string, rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		name := rs.Primary.Attributes["generated_name"]
		if name == fmt.Sprintf("test-basic-%s", rName) {
			return fmt.Errorf("Name should be generated: %s", name)
		}

		return nil
	}
}

func testAccInstance(rName string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver = {
    name = "docker-container"
  }
}
`, rName)
}

func testAccInstanceBootstrapped(rName string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver = {
    name = "docker-container"
  }
  bootstrap = true
}
`, rName)
}

func testAccInstanceGenerated() string {
	return `
resource "buildx_instance" "foo" {
  generate_name = true
  driver = {
    name = "docker-container"
  }
  bootstrap = true
}
`
}
