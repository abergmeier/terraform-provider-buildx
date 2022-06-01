# Frequently Asked Questions

- How to upgrade to latest version of buildx.

  1. Set version you want to upgrade to.
     ```bash
     NEWVER=v0.8.2
     ```
  2. Look at [go.mod] to find used _buildx_ version.
     ```bash
     CURVER=$(go list -m -f '{{ .Version }}' github.com/docker/buildx)
     ```
  3. Fetch latest _buildx_ changes.
     ```bash
     git fetch --all --tags
     ```
  4. Look at whether there were any changes in [commands](commands).
     ```bash
     git log -P ^${CURVER}..${NEWVER} -- '^commands'
     ````
  5. Apply changes from [commands](commands) to Resource code.
