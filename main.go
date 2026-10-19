package main

import "net/http"

func main() {
	client := newEmailClient()
	http.HandleFunc("/signup", signupHandler(client))
	http.ListenAndServe(":8080", nil)
}
