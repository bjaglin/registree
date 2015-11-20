package main

import (
	"fmt"
	"github.com/CenturyLinkLabs/docker-reg-client/registry"
	"github.com/docker/docker/pkg/units"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
)

var (
	client      *registry.Client
	registryURL string
)

type Layer struct {
	id       string
	size     int64
	tags     []string
	children []*Layer
}

func init() {
	client = registry.NewClient()
}

func getRepos() *registry.SearchResults {
	var (
		results *registry.SearchResults
		err     error
	)
	err = retry(3, func() error {
		log.Print("Fetching repos...")
		results, err = client.Search.Query("", 0, 0)
		return err
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%v repo(s) fetched", results.NumResults)
	return results
}

func getTags(name string) registry.TagMap {
	var (
		tags registry.TagMap
		err  error
	)
	err = retry(3, func() error {
		log.Printf("Fetching tags for %s ...", name)
		tags, err = client.Repository.ListTags(name, registry.NilAuth{})
		return err
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%v tags fetched for repo %s", len(tags), name)
	fqTags := make(registry.TagMap)
	for tag, id := range tags {
		fqTags[fqTag(name, tag)] = id
	}
	return fqTags
}

func getAncestry(id string) []string {
	var (
		ancestry []string
		err      error
	)
	err = retry(3, func() error {
		log.Printf("Fetching ancestry for %s ...", id)
		ancestry, err = client.Image.GetAncestry(id, registry.NilAuth{})
		return err
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%v ancestors fetched for tag %s", len(ancestry), id)
	return ancestry
}

func getMetadata(id string) *registry.ImageMetadata {
	var (
		metadata *registry.ImageMetadata
		err      error
	)
	err = retry(3, func() error {
		log.Printf("Fetching metadata for %s ...", id)
		metadata, err = client.Image.GetMetadata(id, registry.NilAuth{})
		return err
	})
	if err != nil {
		panic(err)
	}
	log.Printf("Metadata fetched for tag %s", id)
	return metadata
}

func retry(max int, fn func() error) error {
	var err error
	attempt := 1
	for {
		err = fn()
		attempt++
		if err == nil || attempt > max {
			break
		}
	}
	return err
}

func fqTag(name string, t string) string {
	canonicalName := strings.TrimPrefix(name, "library/")
	return canonicalName + ":" + t
}

func setupRegistryURL() {
	if len(registryURL) == 0 {
		registryURL = os.Getenv("REGISTRY_URL")
	}
	if len(registryURL) == 0 {
		log.Fatal("No registry URL provided, use the environment variable REGISTRY_URL to set it")
	}
	client.BaseURL, _ = url.Parse(registryURL + "/v1/")
}

func getTagsByImage() map[string][]string {
	var (
		wg          sync.WaitGroup
		throttleCh  = make(chan struct{}, 10)    // helper to limit concurrency
		tagsCh      = make(chan registry.TagMap) // tags fetcher/consumer channel
		tagsByImage = make(map[string][]string)  // image ids grouped by tags
	)
	for _, repo := range getRepos().Results {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			throttleCh <- struct{}{}
			tagsCh <- getTags(name)
			<-throttleCh
		}(repo.Name)
	}
	go func() {
		wg.Wait()
		close(tagsCh)
	}()
	for tags, ok := <-tagsCh; ok; tags, ok = <-tagsCh {
		for tag, id := range tags {
			tagsByImage[id] = append(tagsByImage[id], tag)
		}
	}
	return tagsByImage
}

func getImagesAsTree(tagsByImage map[string][]string) (roots []*Layer) {
	var (
		wg         sync.WaitGroup
		throttleCh = make(chan struct{}, 10)            // helper to limit concurrency
		ancestryCh = make(chan []string)                // ancestries fetcher/consumer channel
		images     = make(map[string]*Layer)        // already processed nodes as we are building up the trees
		metadataCh = make(chan *registry.ImageMetadata) // metadata fetcher/consumer channel
	)

	// get ancestries in parallel
	log.Printf("Fetching ancestry for %v images...", len(tagsByImage))
	for imageId := range tagsByImage {
		go func(id string) {
			throttleCh <- struct{}{}
			ancestryCh <- getAncestry(id)
			<-throttleCh
		}(imageId)
	}
	// process them as they arrive until all tagged images have been used
	for len(tagsByImage) != 0 {
		var (
			ancestry     = <-ancestryCh
			previousLayer *Layer
		)
		for _, id := range ancestry {
			if layer, ok := images[id]; ok {
				// we already went up the hierarchy from there, just append a new child
				if previousLayer != nil {
					layer.children = append(layer.children, previousLayer)
				}
				previousLayer = nil
				break
			}
			// register the node in the tree
			layer := &Layer{id: id}
			if tags, ok := tagsByImage[id]; ok {
				layer.tags = tags
				// don't wait for that image's ancestry, we already are going up that one
				delete(tagsByImage, id)
			}
			if previousLayer != nil {
				// this is not a leaf in the tree, so attach its child
				layer.children = []*Layer{previousLayer}
			}
			images[id] = layer
			previousLayer = layer
		}
		if previousLayer != nil {
			// the previous loop didn't break out, so the last node considered is a root
			roots = append(roots, previousLayer)
		}
	}

	// retrieve size of all images
	for id := range images {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			throttleCh <- struct{}{}
			metadataCh <- getMetadata(id)
			<-throttleCh
		}(id)
	}
	go func() {
		wg.Wait()
		close(metadataCh)
	}()
	for metadata, ok := <-metadataCh; ok; metadata, ok = <-metadataCh {
		images[metadata.ID].size = metadata.Size
	}

	return roots
}

func printTree(layer *Layer, level int, cumsize int64) {
	cumsize = cumsize + layer.size
	if len(layer.tags) > 0 || len(layer.children) > 1 {
		fmt.Printf("%s %s%v %s\n", layer.id, strings.Repeat("  ", level), layer.tags, units.HumanSize(float64(cumsize)))
		level = level + 1
		cumsize = 0
	}
	for _, child := range layer.children {
		printTree(child, level, cumsize)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "FATAL ERROR:", r)
		}
	}()

	if len(os.Getenv("REGISTREE_DEBUG")) > 0 {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	setupRegistryURL()

	tagsByImage := getTagsByImage()
	roots := getImagesAsTree(tagsByImage)
	for _, root := range roots {
		printTree(root, 0, 0)
	}

}
