package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	client      *http.Client
	registryURL string
)

type repos struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results"`
	Results    []repo `json:"results"`
}

type repo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type tag string

type image string

type tags map[tag]image

type imageNode struct {
	id       image
	tags     []tag
	children []*imageNode
}

func init() {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: transport}
}

func doGet(path string) []byte {
	res, err := client.Get(registryURL + "/v1/" + path)
	if err != nil {
		panic(err.Error())
	}
	if res.StatusCode != 200 {
		return []byte{}
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}
	res.Body.Close()
	return body
}

func getRepos() repos {
	log.Print("Fetching repos...")
	var repos repos
	json.Unmarshal(doGet("search"), &repos)
	log.Printf("%v repo(s) fetched", repos.NumResults)
	return repos
}

func getTags(name string) map[tag]image {
	log.Printf("Fetching tags for %s ...", name)
	var (
		rawTags tags
		fqTags  tags
	)
	json.Unmarshal(doGet("repositories/"+name+"/tags"), &rawTags)
	log.Printf("%v tags fetched for repo %s", len(rawTags), name)
	fqTags = make(map[tag]image)
	for tag, id := range rawTags {
		fqTags[fqTag(name, tag)] = id
	}
	return fqTags
}

func getAncestry(id image) []image {
	log.Printf("Fetching ancestry for %s ...", id)
	var ancestry []image
	json.Unmarshal(doGet("images/"+string(id)+"/ancestry"), &ancestry)
	log.Printf("%v ancestors fetched for repo %s", len(ancestry), id)
	return ancestry
}

func fqTag(name string, t tag) tag {
	canonicalName := strings.TrimPrefix(name, "library/")
	return tag(canonicalName + ":" + string(t))
}

func printTree(root *imageNode, level int) {
	if len(root.tags) > 0 || len(root.children) > 1 {
		fmt.Printf("%s %s%v\n", root.id, strings.Repeat("  ", level), root.tags)
		level = level + 1
	}
	for _, child := range root.children {
		printTree(child, level)
	}
}

func main() {
	var (
		remaining   int                          // how many more responses are we waiting from the goroutine?
		tagsCh      = make(chan tags, 4)            // tags fetcher/consumer channel
		tagsByImage = make(map[image][]tag)      // image ids grouped by tags
		ancestryCh  = make(chan []image, 4)         // ancestries fetcher/consumer channel
	)
	if len(registryURL) == 0 {
		registryURL = os.Getenv("REGISTRY_URL")
	}
	if len(registryURL) == 0 {
		log.Fatal("No registry URL provided, use the environment variable REGISTRY_URL to set it")
	}
	if len(os.Getenv("REGISTREE_DEBUG")) > 0 {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(ioutil.Discard)
	}
	// get tags in parallel
	for _, repo := range getRepos().Results {
		remaining = remaining + 1
		go func(name string) { tagsCh <- getTags(name) }(repo.Name)
	}
	// group them as they are fetched
	for remaining != 0 {
		for tag, id := range <-tagsCh {
			tags, _ := tagsByImage[id]
			tagsByImage[id] = append(tags, tag)
		}
		remaining = remaining - 1
	}
	// get ancestries in parallel
	for imageId := range tagsByImage {
		remaining = remaining + 1
		go func(id image) { ancestryCh <- getAncestry(id) }(imageId)
	}
	// process them as they arrive
	for remaining != 0 {
		for _, id := range <-ancestryCh {
			fmt.Printf("%s\n", id)
		}
		remaining = remaining - 1
	}
}
