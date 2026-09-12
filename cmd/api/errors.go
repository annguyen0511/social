package main

import (
	"log"
	"net/http"
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Internal server error: %s path: %s error: %v", r.Method, r.URL.Path, err)

	msg := "internal server error"
	writeJSONError(w, http.StatusInternalServerError, msg)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Bad request: %s path: %s error: %v", r.Method, r.URL.Path, err)

	writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Conflict: %s path: %s error: %v", r.Method, r.URL.Path, err)

	msg := "resource conflict"
	writeJSONError(w, http.StatusConflict, msg)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Not found: %s path: %s error: %v", r.Method, r.URL.Path, err)

	msg := "not found"
	writeJSONError(w, http.StatusNotFound, msg)
}
