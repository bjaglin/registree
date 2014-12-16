package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}
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

func main() {
	fmt.Printf("Repos: %v\n", getRepos())
	fmt.Printf("Tags for ssp: %v\n", getTags("library/ssp"))
	fmt.Printf("Ancestry for 08d821bf14f3e6aa1b8bf86c6d3d9878c92e9aab95120a68664db20b9fcc2100: %v\n", getAncestry("08d821bf14f3e6aa1b8bf86c6d3d9878c92e9aab95120a68664db20b9fcc2100"))
}
