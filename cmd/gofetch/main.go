package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	var url string = os.Args[1]
	resp, err := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
		fmt.Println("Response Code", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			fmt.Println("Body Error: ", err)
			return
		}

		fmt.Println(string(body))
		// fmt.Println(body)
	} else {
		fmt.Println("Request Error: ", err)
	}
}
