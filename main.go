package main

import (
	"fmt"
	"github.com/CenturyLinkLabs/docker-reg-client/registry"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
)

var (
	client      *registry.Client
	registryURL string
)

type imageNode struct {
	id       string
	tags     []string
	children []*imageNode
}

func init() {
	client = registry.NewClient()
}

func getRepos() *registry.SearchResults {
	log.Print("Fetching repos...")
	results, _ := client.Search.Query("", 0, 0)
	log.Printf("%v repo(s) fetched", results.NumResults)
	return results
}

func getTags(name string) registry.TagMap {
	log.Printf("Fetching tags for %s ...", name)
	tags, _ := client.Repository.ListTags(name, registry.NilAuth{})
	log.Printf("%v tags fetched for repo %s", len(tags), name)
	fqTags := make(registry.TagMap)
	for tag, id := range tags {
		fqTags[fqTag(name, tag)] = id
	}
	return fqTags
}

func getAncestry(id string) []string {
	log.Printf("Fetching ancestry for %s ...", id)
	ancestry, _ := client.Image.GetAncestry(id, registry.NilAuth{})
	log.Printf("%v ancestors fetched for repo %s", len(ancestry), id)
	return ancestry
}

func fqTag(name string, t string) string {
	canonicalName := strings.TrimPrefix(name, "library/")
	return canonicalName + ":" + t
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
		remaining   int                           // how many more responses are we waiting from the goroutine?
		tagsCh      = make(chan registry.TagMap)  // tags fetcher/consumer channel
		tagsByImage = make(map[string][]string)   // image ids grouped by tags
		ancestryCh  = make(chan []string)         // ancestries fetcher/consumer channel
		images      = make(map[string]*imageNode) // already processed nodes as we are building up the trees
		roots       []*imageNode                  // roots as we are building up the threes
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
	client.BaseURL, _ = url.Parse(registryURL + "/v1/")
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
		go func(id string) { ancestryCh <- getAncestry(id) }(imageId)
	}
	// process them as they arrive until all tagged images have been used
	for len(tagsByImage) != 0 {
		var (
			ancestry     = <-ancestryCh
			previousNode *imageNode
		)
		for _, id := range ancestry {
			if node, ok := images[id]; ok {
				// we already went up the hierarchy from there, just append a new child
				if previousNode != nil {
					node.children = append(node.children, previousNode)
				}
				previousNode = nil
				break
			}
			node := &imageNode{id: id}
			if tags, ok := tagsByImage[id]; ok {
				node.tags = tags
				// don't wait for that image's ancestry, we already are going up that one
				delete(tagsByImage, id)
			}
			if previousNode != nil {
				// this is not a leaf in the tree, so attach its child
				node.children = []*imageNode{previousNode}
			}
			images[id] = node
			previousNode = node
		}
		if previousNode != nil {
			// the previous loop didn't break out, so the last node considered is a root
			roots = append(roots, previousNode)
		}
	}
	// dump all the trees
	for _, root := range roots {
		printTree(root, 0)
	}

}
