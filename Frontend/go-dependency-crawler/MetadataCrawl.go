package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type module struct {
	Path    string `json:"Path" bson:"Path"`
	Version string `json:"Version" bson:"Version"`
}

type dep struct {
	Path      string `json:"Path" bson:"Path"`
	Version   string `json:"Version" bson:"Version"`
	CacheTime string `json:"Timestamp" bson:"CacheTime"`
	Mod       struct {
		ModulePath   string   `bson:"ModulePath"`
		GoVersion    string   `bson:"GoVersion"`
		DirRequire   []module `bson:"DirRequire"`
		IndirRequire []module `bson:"IndirRequire"`
		Exclude      []module `bson:"Exclude"`
		Replace      []string `bson:"Replace"`
		Retract      []string `bson:"Retract"`
	} `bson:"mod"`
	HasValidMod int    `bson:"HasValidMod"`
	ModFile     string `json:"modFile" bson:"ModFile"`
	IsOnPkg     bool   `bson:"IsOnPkg"`
}

var numOfThread int
var muxThread, muxDB sync.Mutex

func expHandler(cacheTime string, e error) {
	logFile, err := os.OpenFile("./err.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	log.Println(cacheTime, e)
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

func parse(modInfo []dep, client *http.Client, collection *mongo.Collection) {
	fmt.Println(modInfo[0].Path, modInfo[0].CacheTime)
	for index, val := range modInfo {
		if index == 0 {
			continue
		}
		resp, err := client.Get("https://proxy.golang.org/" + trans(val.Path) + "/@v/" + trans(val.Version) + ".mod")
		modInfo[index].HasValidMod = 1
		var modtext []byte
		var lines []string
		if err != nil {
			expHandler(modInfo[0].CacheTime, err)
			modInfo[index].HasValidMod = -3
		} else {
			if resp.StatusCode != 200 {
				resp, err = client.Get("https://proxy.golang.org/" + trans(val.Path) + "/@v/" + trans(val.Version) + ".mod")
				if err != nil {
					expHandler(modInfo[0].CacheTime, err)
					modInfo[index].HasValidMod = -3
				} else {
					modInfo[index].HasValidMod = 0
				}
			}
			if resp != nil {
				modtext, _ = ioutil.ReadAll(resp.Body)
				lines = strings.Split(string(modtext), "\n")
			} else {
				fmt.Println("[ERROR!!!]", modInfo[index])
			}
		}
		modInfo[index].ModFile = string(modtext)

		if modInfo[index].HasValidMod == -3 {
			continue
		}
		if len(lines) > 2 {
			flagList := make([]bool, 4)
			for _, line := range lines {
				words := strings.Fields(line)
				if len(words) == 0 || (len(words) == 1 && words[0] != ")") || words[0] == "" || (words[0][0] == '/' && words[0][1] == '/') {
					continue
				}
				switch words[0] {
				case ")":
					for i := 0; i < 4; i++ {
						flagList[i] = false
					}
				case "module":
					modInfo[index].Mod.ModulePath = words[1]
				case "go":
					modInfo[index].Mod.GoVersion = words[1]
				case "require":
					if words[1] == "(" {
						flagList[0] = true
					} else if words[1] != "()" {
						if len(words) == 3 {
							modInfo[index].Mod.DirRequire = append(modInfo[index].Mod.DirRequire, module{Path: words[1], Version: words[2]})
						} else {
							modInfo[index].Mod.IndirRequire = append(modInfo[index].Mod.IndirRequire, module{Path: words[1], Version: words[2]})
						}
					}
				case "exclude":
					if len(words) > 2 {
						modInfo[index].Mod.Exclude = append(modInfo[index].Mod.Exclude, module{Path: words[1], Version: words[2]})
					}
				case "replace":
					if len(words) > 1 {
						modInfo[index].Mod.Replace = append(modInfo[index].Mod.Replace, strings.Join(words[1:], " "))
					}
				case "retract":
					if len(words) > 1 {
						modInfo[index].Mod.Retract = append(modInfo[index].Mod.Retract, strings.Join(words[1:], " "))
					}
				}
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
		resp, err = client.Head("https://pkg.go.dev/" + val.Path)
		if err != nil {
			time.Sleep(10 * time.Second)
			resp, err = client.Head("https://pkg.go.dev/" + val.Path)
			if err != nil {
				muxThread.Lock()
				numOfThread--
				muxThread.Unlock()
				expHandler(modInfo[0].CacheTime, err)
				return
			}
		}
		modInfo[index].IsOnPkg = resp.StatusCode == 200
		resp.Body.Close()

		modulePath_ := modInfo[index].Mod.ModulePath
		if strings.HasPrefix(modulePath_, "\"") {
			modulePath_ = strings.Trim(modulePath_, "\"")
		}
		if modInfo[index].Path != modulePath_ {
			modInfo[index].HasValidMod = -2
		}
	}

	newValue := make([]interface{}, 0)
	for _, v := range modInfo[1:] {
		newValue = append(newValue, v)
	}

	muxDB.Lock()
	_, err := collection.InsertMany(context.TODO(), newValue)
	if err != nil {
		log.Fatal(err)
	}
	muxDB.Unlock()

	muxThread.Lock()
	numOfThread--
	muxThread.Unlock()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	maxThread := 8

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017").SetConnectTimeout(10*time.Second))
	if err != nil {
		fmt.Print(err)
		return
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	db := client.Database("GoDependency")
	collection := db.Collection("metadata")

	proxyUrl := "http://127.0.0.1:7890"
	proxy, _ := url.Parse(proxyUrl)
	tr := &http.Transport{
		Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second,
	}

	lastModCacheTime := "2019-04-10T19:08:52.997264Z"

	for {
		if lastModCacheTime > "2019-04-10T19:08:52.997264Z" {
			for numOfThread > 0 {
			}
			fmt.Println("\n\n********************", lastModCacheTime)
			return
		}
		resp, err := httpClient.Get("https://index.golang.org/index?since=" + lastModCacheTime)
		if err != nil {
			expHandler(lastModCacheTime, err)
			return
		}
		defer resp.Body.Close()
		var modIndexes []dep
		dec := json.NewDecoder(resp.Body)
		for dec.More() {
			var modIndex dep
			if err := dec.Decode(&modIndex); err != nil {
				expHandler(lastModCacheTime, err)
				for numOfThread > 0 {
				}
				return
			}
			modIndexes = append(modIndexes, modIndex)
		}
		if len(modIndexes) == 1 {
			break
		}
		lastModCacheTime = modIndexes[len(modIndexes)-1].CacheTime

		for numOfThread >= maxThread {
		}
		muxThread.Lock()
		numOfThread++
		muxThread.Unlock()
		go parse(modIndexes, httpClient, collection)
	}

	for numOfThread > 0 {
	}
}
