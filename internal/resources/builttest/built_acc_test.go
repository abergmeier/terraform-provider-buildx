package instancetest

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccBuilt_docker(t *testing.T) {
	t.Parallel()

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
				Config: testAccBuiltDocker(rName, dir),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccBuilt_invalid_type(t *testing.T) {
	t.Parallel()

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
				Config:      testAccBuiltInvalidOutputType(rName, dir),
				Check:       resource.ComposeTestCheckFunc(),
				ExpectError: regexp.MustCompile(`.*Unsupported block type.*`),
			},
		},
	})
}

func TestAccBuilt_local(t *testing.T) {
	t.Parallel()

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
				Config: testAccBuiltLocal(rName, dir),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccBuilt_oci(t *testing.T) {
	t.Parallel()

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
				Config: testAccBuiltOci(rName, dir),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccBuilt_tar(t *testing.T) {
	t.Parallel()

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
				Config: testAccBuiltTar(rName, dir),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAccBuiltDocker(rName, dir string) string {
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
    docker {
      dest = "%s/image.docker"
    }
  }
  instance = buildx_instance.foo.name
}
`, rName, dir)
}

func testAccBuiltInvalidOutputType(rName, dir string) string {
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
    dir {
      dest = "%s"
    }
  }
  depends_on = [
    buildx_instance.foo,
  ]
}
`, rName, dir)
}

func testAccBuiltLocal(rName, dir string) string {
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
    local {
      dest = "%s"
    }
  }
  depends_on = [
    buildx_instance.foo,
  ]
}
`, rName, dir)
}

func testAccBuiltOci(rName, dir string) string {
	return fmt.Sprintf(`
resource "buildx_instance" "foo" {
  name = "test-basic-%s"
  driver {
    name = "docker-container"
  }
  bootstrap = true
}

resource "buildx_built" "foo" {
  instance = buildx_instance.foo.name
  file     = "testdata/Containerfile"
  context  = "."
  output {
    oci {
      dest = "%s/oci.tar"
    }
  }
}
`, rName, dir)
}

func testAccBuiltTar(rName, dir string) string {
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
    tar {
      dest = "%s/test.tar"
    }
  }
  depends_on = [
    buildx_instance.foo,
  ]
}
`, rName, dir)
}
