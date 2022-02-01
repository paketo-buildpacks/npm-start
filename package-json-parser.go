package npmstart

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageScripts struct {
	PostStart string `json:"poststart"`
	PreStart  string `json:"prestart"`
	Start     string `json:"start"`
}

type PackageJson struct {
	Scripts PackageScripts `json:"scripts"`
}

func NewPackageJsonFromPath(filelocation string) (*PackageJson, error) {
	file, err := os.Open(filelocation)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var pkg PackageJson

	err = json.NewDecoder(file).Decode(&pkg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode package.json %w", err)
	}

	return &pkg, nil
}

func (pkg PackageJson) hasStartCommand() bool {
	return pkg.Scripts.Start != ""
}
