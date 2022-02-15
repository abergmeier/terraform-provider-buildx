package instancetest

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccBuilt_basic(t *testing.T) {
	dir, err := ioutil.TempDir("", "acc_built")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccBuilt(rName, dir),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAccBuilt(rName, dir string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver {
    name = "docker-container"
  }
  bootstrap = true
}

resource "buildx_built" "foo" {
  file = "testdata/Containerfile"
  context = "."
  output {
    type = "local"
	dest = "%s"
  }
}
`, rName, dir)
}
