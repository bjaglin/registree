package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	var repos repos
	json.Unmarshal(doGet("search"), &repos)
	return repos
}

func getTags(name string) map[tag]image {
	var tags tags
	json.Unmarshal(doGet("repositories/"+name+"/tags"), &tags)
	return tags
}

func getAncestry(id image) []image {
	var ancestry []image
	json.Unmarshal(doGet("images/"+string(id)+"/ancestry"), &ancestry)
	return ancestry
}

func fqTag(name string, t tag) tag {
	canonicalName := strings.TrimPrefix(name, "library/")
	return tag(canonicalName + ":" + string(t))
}

func printTree(root *imageNode, level int) {
	if len(root.tags) > 0 || len(root.children) > 0 {
		fmt.Printf("%s%v\n", strings.Repeat(" ", level), root.tags)
		level = level +1
	}
	for _, child := range root.children {
		printTree(child, level)
	}
}

func main() {
	var (
		tagsByImage = make(map[image][]tag)
		images      = make(map[image]*imageNode)
		roots       []*imageNode
	)
	for _, repo := range getRepos().Results {
		name := repo.Name
		for t, id := range getTags(name) {
			tags, _ := tagsByImage[id]
			tagsByImage[id] = append(tags, fqTag(name, t))
		}
	}
	for imageId := range tagsByImage {
		var (
			root         *imageNode
			previousNode *imageNode
		)
		for _, ancestryId := range getAncestry(imageId) {
			if node, ok := images[ancestryId]; ok {
				if previousNode != nil {
					node.children = append(node.children, previousNode)
				}
				root = node
				break
			}
			node := &imageNode{}
			if tags, ok := tagsByImage[ancestryId]; ok {
				node.tags = tags
			}
			if previousNode != nil {
				node.children = []*imageNode{previousNode}
			}
			images[ancestryId] = node
			previousNode = node
		}
		if root != nil {
			roots = append(roots, root)
		}
	}
	for _, root := range roots {
		printTree(root, 0)
	}

}
