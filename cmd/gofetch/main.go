package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func main() {
	var urls []string = os.Args[1:]
	for _, url := range urls {
		title, err := handleUrl(url)
		if err != nil {
			fmt.Printf("%s %s", title, err)
		} else {
			fmt.Println(title)
		}
	}
}

func handleUrl(url string) (string, error) {
	resp, err := http.Get(url)

	if err != nil {
		return "Request Error: ", err
	}

	defer resp.Body.Close()

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		return "Parse Error: ", err
	}

	title := findTitle(rootNode)
	return title, nil
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
