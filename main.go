package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var users = map[string]string{
	"tong":   "test",
	"kilmou": "test",
}

var sessions = map[string]session{}

type session struct {
	username string
	expiry   time.Time
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

func (s session) isExprired() bool {
	return s.expiry.Before(time.Now())
}

func Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("signin...")
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)

	fmt.Println(err)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println(creds)
	expectedPassword, ok := users[creds.Username]
	if !ok || expectedPassword != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second)
	sessions[sessionToken] = session{
		username: creds.Username,
		expiry:   expiresAt,
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: expiresAt,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println("logout...")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value
	delete(sessions, sessionToken)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	fmt.Println("refresh...")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value
	userSession, exists := sessions[sessionToken]
	fmt.Println(userSession, exists)
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second)
	sessions[newSessionToken] = session{
		username: userSession.username,
		expiry:   expiresAt,
	}
	delete(sessions, sessionToken)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(120 * time.Second),
	})
}

func Welcome(w http.ResponseWriter, r *http.Request) {
	fmt.Println("welcome...")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value
	userSession, exists := sessions[sessionToken]
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if userSession.isExprired() {
		delete(sessions, sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Write([]byte(fmt.Sprintf("Welcome %s!", userSession.username)))
}

func Home(w http.ResponseWriter, r *http.Request) {
	fmt.Println("home...")
	fmt.Fprintf(w, "Home\n")
}

func main() {
	// http.HandleFunc("/headers", headers)
	// http.HandleFunc("/", Home)
	http.HandleFunc("/auth/login", Login)
	http.HandleFunc("/auth/logout", Logout)
	http.HandleFunc("/auth/refresh", Refresh)
	http.HandleFunc("/auth/welcome", Welcome)
	http.ListenAndServe(":8090", nil)
}
