package compose

import "embed"

//go:embed *.yaml
var templates embed.FS

func Template(name string) ([]byte, error) {
	return templates.ReadFile(name)
}
