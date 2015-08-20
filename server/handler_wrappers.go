package main

import (
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

func requireAuthorizedUser(h http.HandlerFunc) http.HandlerFunc {
	h = authorizedReaders(h)
	h = authenticated(h)
	h = applicationScoped(h)
	h = authenticate(h)
	return h
}

func authenticate(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, err := loadUserFromSession(r)
		if err != nil {
			log.Println("error when trying to get current user", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if currentUser == nil {
			currentUser, err = loadUserWithApiToken(r)
			if err != nil {
				log.Println("error when trying to get current user via Api Token", err)
				http.Error(w, "wrong API token", http.StatusInternalServerError)
				return
			}
		}

		if currentUser != nil {
			context.Set(r, CurrentUser, currentUser)
		}

		fn(w, r)
	}
}

func applicationScoped(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		application, err := findApplication(vars["application"])
		if err != nil {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}

		if application != nil {
			context.Set(r, CurrentApplication, application)
		}

		fn(w, r)
	}
}

func authenticated(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser := getCurrentUser(r)
		if currentUser == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		} else {
			fn(w, r)
		}
	}
}

func authorizedReaders(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser := getCurrentUser(r)
		application := getCurrentApplication(r)

		if application.IsReader(currentUser.Name) {
			fn(w, r)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}
