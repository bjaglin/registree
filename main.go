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
		fmt.Printf("%s%v\n", strings.Repeat(" ", level), root.tags)
		level = level + 1
	}
	for _, child := range root.children {
		printTree(child, level)
	}
}

func main() {
	var (
		remaining   int
		tagsCh      = make(chan tags)
		tagsByImage = make(map[image][]tag)
		ancestryCh  = make(chan []image)
		images      = make(map[image]*imageNode)
		roots       []*imageNode
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
		go func(id image) { ancestryCh <- getAncestry(id) }(imageId)
	}
	// process them as they arrive until all tagged images have been used
	for len(tagsByImage) != 0 {
		var (
			ancestry     = <-ancestryCh
			previousNode *imageNode
		)
		for _, id := range ancestry {
			if node, ok := images[id]; ok {
				if previousNode != nil {
					node.children = append(node.children, previousNode)
				}
				previousNode = nil
				break
			}
			node := &imageNode{}
			if tags, ok := tagsByImage[id]; ok {
				node.tags = tags
				delete(tagsByImage, id)
			}
			if previousNode != nil {
				node.children = []*imageNode{previousNode}
			}
			images[id] = node
			previousNode = node
		}
		if previousNode != nil {
			roots = append(roots, previousNode)
		}
	}
	// dump all the trees
	for _, root := range roots {
		printTree(root, 0)
	}

}
