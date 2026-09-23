package main

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type LoginAttempt struct {
	State    string
	Verifier string
	Expires  time.Time
}

type LoginAttemptStore struct {
	mu       sync.Mutex
	attempts map[string]LoginAttempt
}

func (s *LoginAttemptStore) EvictExpired(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.evictExpired()

		case <-ctx.Done():
			return
		}
	}
}

func (s *LoginAttemptStore) evictExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, attempt := range s.attempts {
		if now.After(attempt.Expires) {
			delete(s.attempts, id)
		}
	}
}

func (s *LoginAttemptStore) RegisterAttempt(attemptID string, attempt LoginAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.attempts[attemptID]; ok {
		return errors.New("attempt ID already exists")
	}
	s.attempts[attemptID] = attempt
	return nil
}

func (s *LoginAttemptStore) GetAttempt(attemptID string) (LoginAttempt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt, ok := s.attempts[attemptID]
	return attempt, ok
}

func (s *LoginAttemptStore) DeleteAttempt(attemptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, attemptID)
}

func NewLoginAttemptStore() *LoginAttemptStore {
	return &LoginAttemptStore{
		attempts: make(map[string]LoginAttempt),
	}
}

var discordOAuth = &oauth2.Config{
	ClientID:     mustEnv("DISCORD_CLIENT_ID"),
	ClientSecret: mustEnv("DISCORD_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:8080/api/auth/discord/callback",

	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://discord.com/oauth2/authorize",
		TokenURL:  "https://discord.com/api/v10/oauth2/token",
		AuthStyle: oauth2.AuthStyleInHeader,
	},

	Scopes: []string{"identify"},
}

type AuthStruct struct {
	loginAttempts *LoginAttemptStore
	discordClient *DiscordClient
	db            *DB
}

func NewAuthStruct() (*AuthStruct, error) {
	db, err := NewDB()
	if err != nil {
		return nil, err
	}
	return &AuthStruct{
		loginAttempts: NewLoginAttemptStore(),
		discordClient: NewDiscordClient(),
		db:            db,
	}, nil
}

func (a *AuthStruct) loginDiscord(w http.ResponseWriter, r *http.Request) {
	attemptID, err := randomToken(32)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	verifier := oauth2.GenerateVerifier()

	err = a.loginAttempts.RegisterAttempt(attemptID, LoginAttempt{
		State:    state,
		Verifier: verifier,
		Expires:  time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "discord_login_attempt",
		Value:    attemptID,
		Path:     "/",
		HttpOnly: true,
		//TODO: this
		Secure:   false, // localhost; true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	url := discordOAuth.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
	)

	http.Redirect(w, r, url, http.StatusFound)
}

//TODO: for the love of god add an optional "monad"

type DiscordOAuthUser struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	GlobalName    *string `json:"global_name"`
	Avatar        *string `json:"avatar"`
	Discriminator string  `json:"discriminator"`
}

// TODO: REMOVE TEST
var testServerID = mustEnv("RDM_MAIN_SERVER_ID")
var rdmParticipationRoleID = mustEnv("RDM_MEMBER_ROLE_ID")
var rdmViceAdminRoleID = mustEnv("RDM_VICE_ADMIN_ROLE_ID")
var rdmAdminRoleID = mustEnv("RDM_ADMIN_ROLE_ID")

// Validly returns: Login Succeeded (200), Login Failed Generic (400), Login Attempt Expired (400), Invalid Authentication (403)
func (a *AuthStruct) discordCallback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("discord_login_attempt")
	if err != nil {
		http.Error(w, "missing login attempt", http.StatusBadRequest)
		return
	}

	attempt, ok := a.loginAttempts.GetAttempt(cookie.Value)
	if !ok {
		http.Error(w, "login attempt expired", http.StatusBadRequest)
		return
	}

	// Verify state.
	if !hmac.Equal(
		[]byte(attempt.State),
		[]byte(r.URL.Query().Get("state")),
	) {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	// Exchange Discord's authorization code for tokens.
	token, err := discordOAuth.Exchange(
		r.Context(),
		r.URL.Query().Get("code"),
		oauth2.VerifierOption(attempt.Verifier),
	)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusBadRequest)
		return
	}

	user, err := getDiscordUser(r.Context(), token)
	if err != nil {
		log.Printf("failed to get discord user: %v\n", err)
		http.Error(w, "failed to get discord user", http.StatusInternalServerError)
		return
	}

	member, err := a.discordClient.GetGuildMember(r.Context(), testServerID, user.ID)
	if err != nil {
		http.Error(w, "failed to get guild member", http.StatusInternalServerError)
		return
	}

	// The login attempt has served its purpose.
	a.loginAttempts.DeleteAttempt(cookie.Value)

	if !eligibleForLogin(member) {
		http.Error(w, "not eligible for login", http.StatusForbidden)
		return
	}

	userId, err := a.db.getOrCreateUser(user.ID)
	if err != nil {
		log.Printf("failed to get/create user by discord id: %v\n", err)
		http.Error(w, "failed to get/create user by discord id", http.StatusInternalServerError)
		return
	}

	sessionToken, err := a.db.createSession(userId.ID)
	if err != nil {
		log.Printf("failed to create session token: %v\n", err)
		http.Error(w, "failed to create session token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		//TODO: this
		Secure:   false, // localhost; true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // A week, in seconds
	})
	http.Redirect(w, r, "/", http.StatusFound)
	w.WriteHeader(http.StatusOK)
}

type SessionResponse struct {
	User UserReply `json:"user"`
}

func (a *AuthStruct) getSession(w http.ResponseWriter, r *http.Request) {
	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "nope!", http.StatusForbidden)
		return
	}
	if id == 0 {
		http.Error(w, "failed to get user id", http.StatusInternalServerError)
		return
	}
	usr, err := a.db.getUserByID(id)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if usr == nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	usrReply, err := a.generateUserReply(r.Context(), usr.DiscordID)
	//log.Print("usrReply: ", usrReply)
	//log.Print("userReply img: ", *usrReply.AvatarURL)
	if err != nil {
		log.Printf("failed to generate user reply: %v\n", err)
		http.Error(w, "failed to generate user reply", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(SessionResponse{User: usrReply})
	if err != nil {
		return
	}

}

func (a *AuthStruct) logout(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "no authorization token provided", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to get session token", http.StatusInternalServerError)
		return
	}
	if token.Value == "" {
		http.Error(w, "authorization token is empty", http.StatusBadRequest)
		return
	}

	err = a.db.deleteSession(token.Value)
	if err != nil {
		http.Error(w, "failed to delete session state", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		//TODO: this
		Secure:   false, // localhost; true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	return
}

func (a *AuthStruct) checkAndGetIDAuth(r *http.Request) (int64, error) {
	token, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return 0, errors.New("no authorization token provided")
		}
		return 0, err
	}
	if token.Value == "" {
		return 0, errors.New("no authorization token provided")
	}
	session, err := a.db.getSession(token.Value)
	if err != nil {
		return 0, errors.New("invalid session")
	}
	return session.UserID, nil
}

func getDiscordUser(ctx context.Context, token *oauth2.Token) (*DiscordOAuthUser, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://discord.com/api/v10/users/@me",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user request failed: %s", resp.Status)
	}

	var user DiscordOAuthUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func eligibleForLogin(usr *GuildMember) bool {
	if usr.Roles == nil || len(usr.Roles) == 0 {
		return false
	}
	return slices.Contains(usr.Roles, rdmParticipationRoleID)
}

type isAdminReturn struct {
	IsAdmin bool `json:"isAdmin"`
}

func (a *AuthStruct) isAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		http.Error(w, "nope!", http.StatusForbidden)
		return
	}
	if id == 0 {
		http.Error(w, "failed to get user id", http.StatusInternalServerError)
		return
	}

	usr, err := a.db.getUserByID(id)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	if usr == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if a.discordClient.hasAdminRole(r.Context(), usr.DiscordID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(isAdminReturn{IsAdmin: true})
		if err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(isAdminReturn{IsAdmin: false})
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	return
}

func (a *AuthStruct) checkAndGetIDAuthAdmin(r *http.Request) (int64, error) {
	id, err := a.checkAndGetIDAuth(r)
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("nope")
	}

	usr, err := a.db.getUserByID(id)
	if err != nil {
		return 0, err
	}

	if usr == nil {
		return 0, errors.New("user not found")
	}

	if a.discordClient.hasAdminRole(r.Context(), usr.DiscordID) {
		return id, nil
	}

	return 0, errors.New("not an admin")
}
