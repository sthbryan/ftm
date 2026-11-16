// Command openapi prints the OpenAPI document of ftm's HTTP API. The web's
// generated types (web/src/lib/api/types/openapi.ts) are built from its output
// by `bun run gen:api`.
package main

import (
	"fmt"
	"os"

	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

func main() {
	doc, err := httpapi.OpenAPI()
	if err != nil {
		fmt.Fprintln(os.Stderr, "openapi:", err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(append(doc, '\n'))
}
