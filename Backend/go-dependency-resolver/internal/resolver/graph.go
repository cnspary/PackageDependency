package resolver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/mod/module"
	"math/rand"
	"sync"
)

var muxDB sync.Mutex

// RunGraph initiates the graph construction and processing logic.
func RunGraph(target module.Version, client *mongo.Client, httpClient *http.Client) {
	db := client.Database("dep")
	collection := db.Collection("metadata")

	rand.Seed(time.Now().UnixNano())

	indirectInfoMap := make(map[module.Version]map[module.Version]bool)
	gopathList := make(map[module.Version]bool)

	mg, ok, info, tarInfo := LoadModGraph(target, collection, httpClient, indirectInfoMap, gopathList)
	if !ok {
		fmt.Println("Failed to load module graph.")
		return
	}

	BuildList(mg, info)
}