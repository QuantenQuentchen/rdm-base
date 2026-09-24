package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
)

const baseUrl string = "http://localhost:8080"

func main() {

	portConfig := flag.Int("port", 8080, "port to listen on")
	localConfig := flag.Bool("local", false, "run in local mode")

	port := ":" + strconv.Itoa(*portConfig)

	flag.Parse()

	auth, err := NewAuthStruct(*localConfig)
	if err != nil {
		log.Fatalf("failed to create auth struct: %v", err)
	}

	if auth.local {
		http.Handle("/", http.FileServer(http.Dir("../rdm-frontend/dist")))
	} else {
		http.Handle("/", http.FileServer(http.Dir("./resources/web")))
	}
	http.HandleFunc("/api/auth/session", auth.getSession)

	http.HandleFunc("/api/auth/discord/login", auth.loginDiscord)
	http.HandleFunc("/api/auth/discord/callback", auth.discordCallback)

	http.HandleFunc("/api/auth/logout", auth.logout)

	http.HandleFunc("/api/auth/admin", auth.isAdmin)

	http.HandleFunc("/api/awards/categories", auth.handleGetCategories)

	http.HandleFunc("/api/awards/categories/{categoryID}/suggestions", auth.handleSuggestionUpsert)

	http.HandleFunc("/api/awards/suggestions/mine", auth.handleMineSuggestions)

	http.HandleFunc("/api/admin/viewSuggestions", auth.handleViewSuggestions)
	http.HandleFunc("/api/admin/upsert/Categories", auth.handleCategoryUpsert)
	http.HandleFunc("/api/admin/delete/Categories", auth.handleCategoryDeletion)

	log.Println("Listening on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
