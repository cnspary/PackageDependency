package resolver

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	//"time"

	"github.com/yangshenyi/ymodule/mymvs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	//"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"net/http"
	/*
		"crypto/tls"
		"net/url"
	*/)

type Version struct {
	Path    string `json:"Path" bson:"Path"`
	Version string `json:"Version" bson:"Version"`
}

type ModInfo struct {
	Path    string `json:"Path" bson:"Path"`
	Version string `json:"Version" bson:"Version"`
	Mod     struct {
		ModulePath   string    `bson:"ModulePath"`
		GoVersion    string    `bson:"GoVersion"`
		DirRequire   []Version `bson:"DirRequire"`
		IndirRequire []Version `bson:"IndirRequire"`
		Exclude      []Version `bson:"Exclude"`
		Replace      []string  `bson:"Replace"`
		Retract      []string  `bson:"Retract"`
	} `bson:"mod"`
	HasValidMod int    `json:"HasValidMod" bson:"HasValidMod"`
	ModFile     string `bson:"ModFile"`
}

func pruningForGoVersion(goVersion string) bool {
	return semver.Compare("v"+goVersion, "v1.17") >= 0
}

func cmpVersion(v1, v2 string) int {
	if v2 == "" {
		if v1 == "" {
			return 0
		}
		return -1
	}
	if v1 == "" {
		return 1
	}
	return semver.Compare(v1, v2)
}

func replacement(mod module.Version, replace map[module.Version]module.Version) (fromVersion string, to module.Version) {
	if r, ok := replace[mod]; ok {
		return mod.Version, r
	}
	if r, ok := replace[module.Version{Path: mod.Path, Version: ""}]; ok {
		return "", r
	}
	return "", mod
}

func ParseModFile(modFile string, recv *ModInfo) {
	lines := strings.Split(string(modFile), "\n")
	if len(lines) < 2 {
		return
	}
	flags := make([]bool, 3)

	for _, line := range lines {
		var words []string = strings.Fields(line)
		//fmt.Println(ModInfo[index])
		if len(words) == 0 || words[0] == "" || words[0][0] == '/' && words[0][1] == '/' {
			continue
		} else if words[0] == "module" {
			recv.Mod.ModulePath = words[1]
		} else if words[0] == "go" {
			recv.Mod.GoVersion = words[1]
		} else if words[0] == ")" {
			for i := 0; i < len(flags); i++ {
				flags[i] = false
			}
		} else if flags[0] {
			if len(words) == 1 {
				log.Fatal("resolve fail[11]")
				break
			} else if len(words) == 2 {
				recv.Mod.DirRequire = append(recv.Mod.DirRequire, Version{Path: words[0], Version: words[1]})
			} else {
				recv.Mod.IndirRequire = append(recv.Mod.IndirRequire, Version{Path: words[0], Version: words[1]})
			}
		} else if flags[1] {
			if len(words) < 2 {
				log.Fatal("resolve fail[22]")
				break
			}
			recv.Mod.Exclude = append(recv.Mod.Exclude, Version{Path: words[0], Version: words[1]})
		} else if flags[2] {
			if len(words) == 1 {
				log.Fatal("resolve fail[33]")
				break
			} else if len(words) == 2 {
				recv.Mod.DirRequire = append(recv.Mod.DirRequire, Version{Path: words[0], Version: words[1]})
			} else {
				recv.Mod.IndirRequire = append(recv.Mod.IndirRequire, Version{Path: words[0], Version: words[1]})
			}
		} else if words[0] == "require" {
			if words[1] == "(" {
				flags[0] = true
			} else if words[1] == "()" {
				continue
			} else {
				if len(words) < 3 {
					log.Fatal("resolve fail[1]")
					break
				}
				if len(words) == 3 {
					recv.Mod.DirRequire = append(recv.Mod.DirRequire, Version{Path: words[1], Version: words[2]})
				} else {
					recv.Mod.IndirRequire = append(recv.Mod.IndirRequire, Version{Path: words[1], Version: words[2]})
				}
			}
		} else if words[0] == "exclude" {
			if len(words) <= 1 {
				log.Fatal("resolve fail[2]")
				break
			} else if words[1] == "(" {
				flags[1] = true
			} else if len(words) == 2 {
				log.Fatal("resolve fail[2]")
				break
			} else {
				recv.Mod.Exclude = append(recv.Mod.Exclude, Version{Path: words[1], Version: words[2]})
			}
		} else if words[0] == "replace" {
			if len(words) <= 1 {
				log.Fatal("resolve fail[3]")
				break
			} else if words[1] == "(" {
				flags[2] = true
			} else {
				recv.Mod.Replace = append(recv.Mod.Replace, strings.Join(words[1:], " "))
			}
		}
	}
}

func trans(src string) string {
	var ret string
	for _, val := range src {
		if val >= 'A' && val <= 'Z' {
			ret += "!" + string(val+'a'-'A')
		} else {
			ret += string(val)
		}
	}
	return ret
}

func readFromFile(m module.Version, recv *ModInfo) {
	module_cache_path := "D:/software_data/goModuleCache/mod"
	module_path := module_cache_path + "/cache/download/" + trans(m.Path) + "/@v/" + trans(m.Version)

	//fmt.Println(module_path)

	f, err := ioutil.ReadFile(module_path + ".mod")
	if err != nil {
		// fmt.Println(m, "GOPATH", err)
	}
	//fmt.Println(string(f))
	ParseModFile(string(f), recv)
}

func getRequiredList(m module.Version, ModInfo ModInfo, excludeInfo map[module.Version]bool, indirect_info_map map[module.Version]map[module.Version]bool) []module.Version {
	var list []module.Version
	list = make([]module.Version, 0)
	hasExclude := len(excludeInfo) > 0

	if ModInfo.Mod.IndirRequire != nil {
		for _, v := range ModInfo.Mod.IndirRequire {
			if v.Path[0] == '"' {
				v.Path = v.Path[1 : len(v.Path)-1]
			}
			if !hasExclude {
				list = append(list, module.Version{Path: v.Path, Version: v.Version})
			} else if _, ok := excludeInfo[module.Version{Path: v.Path, Version: v.Version}]; !ok {
				list = append(list, module.Version{Path: v.Path, Version: v.Version})
			}
		}
	}
	indire_tmp := make(map[module.Version]bool, 0)
	for _, v := range list {
		indire_tmp[v] = true
	}
	indirect_info_map[m] = indire_tmp

	if ModInfo.Mod.DirRequire != nil {
		for _, v := range ModInfo.Mod.DirRequire {
			if v.Path[0] == '"' {
				v.Path = v.Path[1 : len(v.Path)-1]
			}
			if !hasExclude {
				list = append(list, module.Version{Path: v.Path, Version: v.Version})
			} else if _, ok := excludeInfo[module.Version{Path: v.Path, Version: v.Version}]; !ok {
				list = append(list, module.Version{Path: v.Path, Version: v.Version})
			}
		}
	}
	//fmt.Println(len(list))
	module.Sort(list)
	return list
}

func LoadModGraph(target module.Version, collection *mongo.Collection, httpClient *http.Client, indirect_info_map map[module.Version]map[module.Version]bool, gopath_list map[module.Version]bool) (*mymvs.Graph, int, *map[module.Version]module.Version, *ModInfo) {

	// load main module's mod info
	var err error
	var targetModInfo ModInfo = ModInfo{}

	if false {
		readFromFile(target, &targetModInfo)
	} else {
		if err = collection.FindOne(context.TODO(), bson.M{"Path": target.Path, "Version": target.Version}).Decode(&targetModInfo); err != nil {
			//fmt.Println(err, "read main module mod file fail! [1]")
			return nil, -1, nil, nil
		}
	}

	//fmt.Println("success", targetModInfo)

	// parse replace info into a map
	replaceInfo := make(map[module.Version]module.Version, len(targetModInfo.Mod.Replace))
	for _, line := range targetModInfo.Mod.Replace {
		words := strings.Fields(line)
		for ix, v := range words {
			if v == "//" {
				words = words[:ix]
			}
		}
		if words[0] == "//" {
			continue
		} else if len(words) < 3 {
			//fmt.Println("Unknown case!", words)
			continue
		} else if words[1] == "=>" {
			if len(words) == 3 {
				//fmt.Println("Replace by LocalPath [!]")
				replaceInfo[module.Version{Path: words[0], Version: ""}] = module.Version{Path: words[2], Version: ""}
			} else if len(words) == 4 {
				replaceInfo[module.Version{Path: words[0], Version: ""}] = module.Version{Path: words[2], Version: words[3]}
			}
		} else if words[2] == "=>" {
			if len(words) == 4 {
				//fmt.Println("Replace by LocalPath [!]")
				replaceInfo[module.Version{Path: words[0], Version: words[1]}] = module.Version{Path: words[3], Version: ""}
			} else if len(words) == 5 {
				replaceInfo[module.Version{Path: words[0], Version: words[1]}] = module.Version{Path: words[3], Version: words[4]}
			}
		} else {
			fmt.Println("Replace resolve fail")
			return nil, -2, nil, nil
		}
	}
	/*
		fmt.Println("[Debug]: replace normal?")
		for k, v := range replaceInfo {
			fmt.Println(k, v)
		}
	*/
	// parse exclude info into a map
	excludeInfo := make(map[module.Version]bool, len(targetModInfo.Mod.Exclude))
	for _, v := range targetModInfo.Mod.Exclude {
		excludeInfo[module.Version{Path: v.Path, Version: v.Version}] = true
	}
	if targetModInfo.Mod.ModulePath == "" {
		modulePath_ := strings.Fields(targetModInfo.ModFile)[1]
		if modulePath_[0:1] == "\"" {
			modulePath_ = modulePath_[1 : len(modulePath_)-1]
		}
		targetModInfo.Mod.ModulePath = modulePath_
	} else if targetModInfo.Mod.ModulePath[0] == '"' {
		targetModInfo.Mod.ModulePath = targetModInfo.Mod.ModulePath[1 : len(targetModInfo.Mod.ModulePath)-1]
	}
	// construct requirement graph
	mg := mymvs.NewGraph(cmpVersion, []module.Version{{Path: targetModInfo.Mod.ModulePath, Version: ""}})
	// does main module enable prune ?
	pruning := pruningForGoVersion(targetModInfo.Mod.GoVersion)
	// add main module's explicit requirements into dependency graph
	roots_ := getRequiredList(module.Version{Path: targetModInfo.Mod.ModulePath, Version: ""}, targetModInfo, excludeInfo, indirect_info_map)
	var roots []module.Version
	for _, v := range roots_ {
		if v.Path != targetModInfo.Mod.ModulePath {
			roots = append(roots, v)
		}
	}
	module.Sort(roots)
	//fmt.Println(roots, targetModInfo)
	mg.Require(module.Version{Path: targetModInfo.Mod.ModulePath, Version: ""}, roots)

	// has a module been expanded?
	expandedQueue := make(map[module.Version]bool, len(roots))
	// queue of modules waiting for expanding
	//expandingQueue := make(map[module.Version]bool, len(roots))
	// load transitive dependency
	// add successor nodes of selected node with replace and exclude applied

	flagReplaceLocalError := false
	flagOtherError := false

	loadOne := func(m module.Version) ([]module.Version, bool, error) {
		//fmt.Println(m)
		_, actual := replacement(m, replaceInfo)

		// load current module's mod info
		var currentModInfo ModInfo = ModInfo{}

		//fmt.Println("*****", m, "&&", actual)
		if actual.Version != "" {
			if false {
				readFromFile(actual, &currentModInfo)
			} else {
				if err = collection.FindOne(context.TODO(), bson.M{"Path": actual.Path, "Version": actual.Version}).Decode(&currentModInfo); err != nil {
					//fmt.Println(err, "read current module", m.Path, m.Version, "=>", actual.Path, actual.Version, "mod file fail! [1]")
					flagOtherError = true
					return nil, false, err
				}
			}
			// resolve local path
		} else {
			var finalPath string = ""
			if filepath.IsAbs(actual.Path) {
				//fmt.Println("abs")
				flagOtherError = true
				return nil, false, errors.New("!")
			} else {
				//fmt.Println(target.Path, actual.Path)
				urlPath := target.Path
				words := strings.Split(urlPath, "/")
				/*
					//set proxy
					proxyUrl := "http://127.0.0.1:7890"
					proxy, _ := url.Parse(proxyUrl)
					tr := &http.Transport{
						Proxy:           http.ProxyURL(proxy),
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					}
					httpClient := &http.Client{
						Transport: tr,
						Timeout:   time.Second * 120,
					}
				*/

				if words[0] != "github.com" {
					// 还可以在 pkg 上查一下，获取 github 仓库 url
					//fmt.Println("??????")
					flagReplaceLocalError = true
					return nil, false, errors.New("!")
				}

				// consider target virtual Path
				flagVirtual := true
				if words[len(words)-1][0] == 'v' {
					for _, v := range words[len(words)-1][1:] {
						if v < '0' && v > '9' {
							flagVirtual = false
							break
						}
					}
				} else {
					flagVirtual = false
				}

				if flagVirtual {
					urlPath = strings.Join(words[:len(words)-1], "/")
				}

				finalPath = strings.Join(strings.Split(filepath.Join(urlPath, actual.Path), "\\"), "/")
				//fmt.Println(flagVirtual, finalPath)

				var finalVersion string
				if v := strings.Split(target.Version, "-"); len(v) > 2 {
					finalVersion = v[len(v)-1]
				} else {
					// submodule label?
					labelPrefix := ""
					if v := strings.Split(urlPath, "/"); len(v) > 3 {
						labelPrefix = strings.Join(v[3:], "/")
					}
					//fmt.Println(labelPrefix)
					if v := strings.Split(target.Version, "+"); len(v) > 1 {
						finalVersion = labelPrefix + v[0]
					} else {
						finalVersion = labelPrefix + "/" + target.Version
					}
				}
				//fmt.Println(finalVersion)

				//fmt.Println("https://raw.githubusercontent.com/" + words[1] + "/" + words[2] + "/" + finalVersion + "/" + strings.Join(strings.Split(finalPath, "/")[3:], "/") + "/go.mod")
				//fmt.Println(target, finalPath)
				if len(strings.Split(finalPath, "/")) <= 2 {
					flagOtherError = true
					return nil, false, errors.New("!")
				}
				resp, err := httpClient.Get("https://raw.githubusercontent.com/" + words[1] + "/" + words[2] + "/" + finalVersion + "/" + strings.Join(strings.Split(finalPath, "/")[3:], "/") + "/go.mod")
				// consider subdirectory
				if flagVirtual && resp.StatusCode != 200 {
					finalPath = strings.Join(strings.Split(filepath.Join(target.Path, actual.Path), "\\"), "/")
					resp, err = httpClient.Get("https://raw.githubusercontent.com/" + words[1] + "/" + words[2] + "/" + finalVersion + "/" + strings.Join(strings.Split(finalPath, "/")[3:], "/") + "/go.mod")
				}
				if err != nil {
					// fmt.Println("Internet error!")
					flagOtherError = true
					return nil, false, errors.New("!")
				}
				modFile, _ := ioutil.ReadAll(resp.Body)
				//fmt.Println(string(modFile))
				ParseModFile(string(modFile), &currentModInfo)
			}
		}
		if currentModInfo.HasValidMod == 0 {
			gopath_list[m] = true
		}
		if reqs, ok := mg.RequiredBy(m); !ok {
			requiredList := getRequiredList(m, currentModInfo, excludeInfo, indirect_info_map)
			//fmt.Println(requiredList)
			mg.Require(m, requiredList)
			return requiredList, pruningForGoVersion(currentModInfo.Mod.GoVersion), nil
		} else {
			return reqs[:len(reqs):len(reqs)], pruningForGoVersion(currentModInfo.Mod.GoVersion), nil
		}

	}

	var enqueue func(m module.Version, pruning bool)
	enqueue = func(m module.Version, pruning bool) {
		if m.Version == "none" {
			return
		} else if v := strings.Split(target.Version, "+"); v[len(v)-1] == "incompatible" {
			return
		}

		if !pruning {
			if _, ok := expandedQueue[m]; ok {
				return
			}
		}

		requireList, curPruning, err := loadOne(m)
		//fmt.Println("[>]", m, requireList)
		if err != nil {
			flagOtherError = true
			return
		}

		if !pruning || !curPruning {
			nextPruning := curPruning
			if !pruning {
				nextPruning = false
			}
			expandedQueue[m] = true
			for _, r := range requireList {
				/*if r.Path == "gihub.com/go-kit/kit@v0.12.0" {{
					fm.Println(m, r)
				}*/
				enqueue(r, nextPruning)
			}
		}

	}
	//fmt.Println("debug", roots)
	for _, m := range roots {
		enqueue(m, pruning)
	}
	if flagReplaceLocalError {
		return mg, -1, &replaceInfo, &targetModInfo
	} else if flagOtherError {
		return mg, -2, &replaceInfo, &targetModInfo
	}

	return mg, 1, &replaceInfo, &targetModInfo
}
