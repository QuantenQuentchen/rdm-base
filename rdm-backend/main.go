package main

import (
	"log"
	"net/http"
)

func main() {
	auth, err := NewAuthStruct()
	if err != nil {
		log.Fatalf("failed to create auth struct: %v", err)
	}

	http.Handle("/", http.FileServer(http.Dir("../rdm-frontend/dist")))

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

	log.Println("Listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
