package main

import (
	"net/http"
	_ "net/http/pprof"
)

func Pprof() {

	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()
}
