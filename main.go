package main

import (
	"fmt"
	"net/http"
	"github.com/go-chi/chi/v5"
)

func main() {
	server := &http.Server{
		Addr: ":8080", 
		Handler: http.HandlerFunc(basicHandler),
	}
	fmt.Print("server running....")
	err := server.ListenAndServe()
	if err != nil{
		fmt.Println("failed to listen to server", err)
	}
}

func basicHandler(w http.ResponseWriter, r *http.Request){

}
