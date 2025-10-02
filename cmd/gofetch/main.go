package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type result struct {
	url   string
	title string
	links []string
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

	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			ok := visited.seenUrl(u)
			if !ok {
				visited.handleUrl(u, results)
			}
		}(u)
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
			for _, link := range res.links {
				fmt.Printf("Link: %s\n", link)
			}
		}
	}
}

func (v *Visited) handleUrl(u string, results chan result) {
	resp, err := http.Get(u)

	if err != nil {
		results <- result{url: u, title: "Request Error: ", err: err}
		return
	}

	defer resp.Body.Close()

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		results <- result{url: u, title: "Parse Error: ", err: err}
		return
	}

	title := findTitle(rootNode)
	links := v.findLinks(rootNode, []string{}, &u)
	results <- result{url: u, title: title, links: links, err: nil}
}

func findTitle(node *html.Node) string {

	if node.Type == html.ElementNode && node.Data == "title" {
		if node.FirstChild != nil {
			return node.FirstChild.Data
		}
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

func (v *Visited) findLinks(node *html.Node, links []string, baseUrl *string) []string {
	base, _ := url.Parse(*baseUrl)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "a" && len(links) < 10 {
			for _, attr := range child.Attr {
				if attr.Key == "href" {
					u, err := url.Parse(attr.Val)
					if err != nil || attr.Val == "" || strings.HasPrefix(attr.Val, "#") || strings.HasPrefix(attr.Val, "mailto:") || strings.HasPrefix(attr.Val, "javascript:") || strings.HasPrefix(attr.Val, "tel:") {
						continue
					}
					link := base.ResolveReference(u).String()
					if v.seenUrl(link) {
						continue
					} else {

					}
					links = append(links, link)
				}
			}
		}
		links = v.findLinks(child, links, baseUrl)
		if len(links) > 10 {
			break
		}
	}
	return links
}

func (v *Visited) seenUrl(u string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, ok := v.seen[u]
	if !ok {
		v.seen[u] = true
	}
	return ok
}
