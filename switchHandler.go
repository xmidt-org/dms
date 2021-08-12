package main

import "net/http"

type SwitchHandler struct {
	Switch *Switch
}

func (sh SwitchHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	pr := PostponeRequest{
		Source: request.RemoteAddr,
	}

	if sh.Switch.Postpone(pr) {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}
