package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func main() {
	var urls []string = os.Args[1:]
	c := make(chan string)
	cerr := make(chan error)
	for _, url := range urls {
		go handleUrl(url, c, cerr)
		title := <-c
		err := <-cerr
		if err != nil {
			fmt.Printf("%s %s", title, err)
		} else {
			fmt.Println(title)
		}
	}

}

func handleUrl(url string, c chan string, cerr chan error) {
	resp, err := http.Get(url)

	if err != nil {
		c <- "Request Error: "
		cerr <- err
	}

	defer resp.Body.Close()

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		c <- "Parse Error: "
		cerr <- err
	}

	c <- findTitle(rootNode)
	cerr <- nil
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
