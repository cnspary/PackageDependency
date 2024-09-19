package main

import (
	"fmt"
	"os"

	"go-dependency-crawler/internal/config"
	"go-dependency-crawler/internal/crawler"
	"go-dependency-crawler/internal/db"
	"golang.org/x/mod/module"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Illegal number of command parameters!")
		return
	}

	client := db.ConnectMongoDB(config.GetMongoURI())
	defer db.DisconnectMongoDB(client)

	httpClient := config.GetHTTPClient()

	target := module.Version{Path: os.Args[1], Version: os.Args[2]}
	crawler.RunGraph(target, client, httpClient)
}