package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func main() {
	var url string = os.Args[1]
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Request Error", err)
		return
	}

	defer resp.Body.Close()
	fmt.Println("Response Code", resp.StatusCode)

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		fmt.Println("Parse Error: ", err)
		return
	}

	title := findTitle(rootNode)
	fmt.Println(title)
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
