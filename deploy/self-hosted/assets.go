// Package selfhost supplies the setup helper used by the Cully CLI.
package selfhost

import _ "embed"

// SetupScript runs against the downloaded release's Compose files.
//
//go:embed setup.sh
var SetupScript string
