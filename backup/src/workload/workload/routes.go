package main

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"LoadGet",
		"GET",
		"/load",
		LoadGet,
	},
	Route{
		"LoadSet",
		"POST",
		"/set_load",
		LoadSet,
	},
}
