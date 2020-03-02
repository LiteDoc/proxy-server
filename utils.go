package main

import (
	"log"
	"net/url"
)

// BuildQuery : Take a base URL and variadic string slices of query parameters, return URL with query params
func BuildQuery(baseURL string, queryParams ...[]string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	params := url.Values{}
	for _, queryParam := range queryParams {
		params.Add(queryParam[0], queryParam[1])
	}
	base.RawQuery = params.Encode()
	return base.String()
}

// JSONResponse : Basic response
type JSONResponse struct {
	Status string
	Code   int
}
