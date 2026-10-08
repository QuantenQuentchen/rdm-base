package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type getCategoriesRequest struct {
	Categories []Category `json:"categories"`
}

func (a *AuthStruct) handleGetCategories(w http.ResponseWriter, _ *http.Request) {
	categories, err := a.db.getCategories()
	if err != nil {
		http.Error(w, "failed to get categories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(getCategoriesRequest{
		Categories: categories,
	})
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

type inputSuggestionRequest struct {
	Suggestions []NominationSuggestion `json:"suggestions"`
}

type categorySuggestionsResponse struct {
	Suggestions []NominationSuggestionRich `json:"suggestions"`
}

func (a *AuthStruct) handleCategorySuggestions(w http.ResponseWriter, r *http.Request) {
	_, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}

	categoryID, err := strconv.ParseInt(r.PathValue("categoryID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	if _, err := getCategory(a.db.db, categoryID); err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get category", http.StatusInternalServerError)
		return
	}

	suggestions, err := a.db.getAllSuggestionsForCategory(categoryID)
	if err != nil {
		http.Error(w, "failed to get suggestions", http.StatusInternalServerError)
		return
	}

	suggestionsRich := make([]NominationSuggestionRich, 0, len(suggestions))
	userReplies := make(map[int64]UserReply)
	for _, suggestion := range suggestions {
		userReply, ok := userReplies[suggestion.NominatorID]
		if !ok {
			user, err := a.db.getUserByID(suggestion.NominatorID)
			if err != nil {
				http.Error(w, "failed to get user by ID", http.StatusInternalServerError)
				return
			}
			if user == nil {
				http.Error(w, "user not found", http.StatusInternalServerError)
				return
			}

			userReply, err = a.generateUserReply(r.Context(), user.DiscordID)
			if err != nil {
				http.Error(w, "failed to generate user reply", http.StatusInternalServerError)
				return
			}
			userReplies[suggestion.NominatorID] = userReply
		}

		suggestionsRich = append(suggestionsRich, NominationSuggestionRich{
			Suggestion: suggestion,
			User:       userReply,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categorySuggestionsResponse{
		Suggestions: suggestionsRich,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (a *AuthStruct) handleSuggestionUpsert(w http.ResponseWriter, r *http.Request) {

	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}

	categoryIDStr := r.PathValue("categoryID")

	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	var req inputSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var suggestions []NominationSuggestion
	for _, suggestion := range req.Suggestions {
		suggestions = append(suggestions, NominationSuggestion{
			CategoryID:  categoryID,
			Text:        suggestion.Text,
			UpdatedAt:   time.Now(),
			NominatorID: id,
		})
	}
	err = a.db.upsertSuggestions(suggestions, categoryID, id)
	if err != nil {
		http.Error(w, "failed to upsert suggestions", http.StatusInternalServerError)
		return
	}
	newSuggestions, err := a.db.getSuggestionsFor(categoryID, id)
	if err != nil {
		http.Error(w, "failed to get suggestions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encodeErr := json.NewEncoder(w).Encode(inputSuggestionRequest{
		Suggestions: newSuggestions,
	})
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *AuthStruct) handleMineSuggestions(w http.ResponseWriter, r *http.Request) {
	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}

	userSuggestions, err := a.db.getSuggestionsOf(id)
	if err != nil {
		http.Error(w, "failed to get suggestions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encodeErr := json.NewEncoder(w).Encode(inputSuggestionRequest{
		Suggestions: userSuggestions,
	})
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	//w.WriteHeader(http.StatusOK)

}

//================
// Admin Endpoints
//================

type CategoryUpsertRequest struct {
	ID       *int64   `json:"id"`
	Category Category `json:"category"`
}

func (a *AuthStruct) handleCategoryUpsert(w http.ResponseWriter, r *http.Request) {
	_, err := a.checkAndGetIDAuthAdmin(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}
	var req CategoryUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ID != nil {
		req.Category.ID = *req.ID
	}
	err = a.db.upsertCategory(req.Category)
	if err != nil {
		http.Error(w, "failed to upsert category", http.StatusInternalServerError)
		return
	}
}

/*
	type CategoryReorderRequest struct {
		Order map[int64]int64 `json:"order"`
	}

	func (a *AuthStruct) handleCategoryReorder(w http.ResponseWriter, r *http.Request) {
		_, err := a.checkAndGetIDAuthAdmin(r)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusForbidden)
			return
		}
		var req CategoryReorderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Order == nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		err = a.db.reorderCategories(req.Order)
		if err != nil {
			http.Error(w, "failed to reorder categories", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
*/
type CategoryDeletionRequest struct {
	ID int64 `json:"id"`
}

func (a *AuthStruct) handleCategoryDeletion(w http.ResponseWriter, r *http.Request) {
	_, err := a.checkAndGetIDAuthAdmin(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}
	var req CategoryDeletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err = a.db.deleteCategory(req.ID)
	if err != nil {
		http.Error(w, "failed to delete category", http.StatusInternalServerError)
		return
	}
}

type NominationSuggestionRich struct {
	Suggestion NominationSuggestion `json:"suggestion"`
	User       UserReply            `json:"user"`
}

type ViewAllReply struct {
	Suggestions []NominationSuggestionRich `json:"suggestions"`
}

func (a *AuthStruct) handleViewSuggestions(w http.ResponseWriter, r *http.Request) {
	_, err := a.checkAndGetIDAuthAdmin(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusForbidden)
		return
	}

	var suggestions []NominationSuggestion
	var suggestionsRich []NominationSuggestionRich

	err = a.db.db.Select(
		&suggestions,
		`SELECT * FROM suggestions`,
	)
	if err != nil {
		http.Error(w, "failed to view suggestions", http.StatusInternalServerError)
		return
	}
	for _, suggestion := range suggestions {
		userID, err := a.db.getUserByID(suggestion.NominatorID)
		if err != nil {
			http.Error(w, "failed to get user by ID", http.StatusInternalServerError)
			return
		}
		if userID == nil {
			log.Println("user not found")
			continue
		}
		usrReply, genErr := a.generateUserReply(r.Context(), userID.DiscordID)
		if genErr != nil {
			http.Error(w, "failed to generate user reply"+genErr.Error(), http.StatusInternalServerError)
			return
		}
		suggestionsRich = append(suggestionsRich, NominationSuggestionRich{
			Suggestion: suggestion,
			User:       usrReply,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encodeErr := json.NewEncoder(w).Encode(ViewAllReply{
		Suggestions: suggestionsRich,
	})
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
