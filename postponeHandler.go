package main

import "net/http"

type PostponeHandler struct {
	Postponer Postponer
}

func (ph PostponeHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	pr := PostponeRequest{
		Source: request.RemoteAddr,
	}

	if ph.Postponer.Postpone(pr) {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}
