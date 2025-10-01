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

func main() {
	var urls []string = os.Args[1:]
	results := make(chan result, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			handleUrl(u, results)
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {

		if res.err != nil {
			fmt.Printf("%s %s", res.title, res.err)
		} else {
			fmt.Println(res.title)
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
