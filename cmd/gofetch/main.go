package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"golang.org/x/net/html"
)

type result struct {
	url   string
	title string
	err   error
}

type Visited struct {
	mu   sync.Mutex
	seen map[string]bool
}

func main() {
	var urls []string = os.Args[1:]
	results := make(chan result, len(urls))
	var wg sync.WaitGroup
	visited := &Visited{seen: make(map[string]bool)}

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			ok := visited.seenUrl(u)
			if !ok {
				handleUrl(u, results)
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {

		if res.err != nil {
			fmt.Printf("ERROR: [Title]: %s [Error]: %s", res.title, res.err)
		} else {
			fmt.Printf("[%s]: %s\n", res.url, res.title)
		}
	}
}

func handleUrl(url string, results chan result) {
	resp, err := http.Get(url)

	if err != nil {
		results <- result{url: url, title: "Request Error: ", err: err}
		return
	}

	defer resp.Body.Close()

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		results <- result{url: url, title: "Parse Error: ", err: err}
		return
	}

	results <- result{url: url, title: findTitle(rootNode), err: nil}
}

func findTitle(node *html.Node) string {

	if node.Type == html.ElementNode && node.Data == "title" {
		return node.FirstChild.Data
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			result := findTitle(child)
			if result != "" {
				return result
			}
		}
	}
	return ""
}

func (v *Visited) seenUrl(url string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, ok := v.seen[url]
	if !ok {
		v.seen[url] = true
	}
	return ok
}
