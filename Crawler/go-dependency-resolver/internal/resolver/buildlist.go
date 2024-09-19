package resolver

import (
	"fmt"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// BuildList processes and prints the module build list.
func BuildList(mg *module.Graph, info *map[module.Version]module.Version) {
	buildList := mg.BuildList()
	fmt.Println(buildList[0].Path)
	for _, v := range buildList[1:] {
		if k, ok := (*info)[v]; ok {
			fmt.Println(v.Path, v.Version, "=>", k.Path, k.Version)
		} else if k, ok := (*info)[module.Version{Path: v.Path, Version: ""}]; ok {
			fmt.Println(v.Path, v.Version, "=>", k.Path, k.Version)
		} else {
			fmt.Println(v.Path, v.Version)
		}
	}
}