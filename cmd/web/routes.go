package main

import (
	"go-ws-basics/internal/handlers"
	"net/http"

	"github.com/bmizerany/pat"
)

//func routes to create routes in a webserver using pat
func routes() http.Handler {
	mux := pat.New()

	mux.Get("/", http.HandlerFunc(handlers.Home))
	mux.Get("/ws", http.HandlerFunc(handlers.WsEndpoint))

	//to serve static content
	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Get("/static/", http.StripPrefix("/static", fileServer))
	return mux
}
