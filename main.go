package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/google/uuid"
)

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

type Session struct {
	User   string    `json:"user"`
	Expiry time.Time `json:"expiry"`
}

var (
	users      []User
	sessions   = map[string]Session{}
	expireTime = 60 * time.Minute
)

func Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handler:login")
	// TODO: check if already logged in
	// c, err := r.Cookie("session_token")
	r.ParseForm()
	// fmt.Println(r.Form.Has("name"))
	// fmt.Println(r.Form.Get("name"))
	i := slices.IndexFunc(users, func(u User) bool {
		return r.Form.Get("name") == u.Name && r.Form.Get("password") == u.Password
	})
	if i == -1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user := users[i]
	// fmt.Println("index=", i)
	// fmt.Println("user=", users[i])
	token := uuid.NewString()
	expiresAt := time.Now().Add(expireTime)
	sessions[token] = Session{user.Name, expiresAt}
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expiresAt,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handler:logout")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := c.Value
	delete(sessions, token)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handler:refresh")
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := c.Value
	session, exists := sessions[token]
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	newToken := uuid.NewString()
	expiresAt := time.Now().Add(expireTime)
	sessions[newToken] = Session{session.User, expiresAt}
	delete(sessions, token)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newToken,
		Expires: expiresAt,
	})
}

func Status(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handler:status")
	fmt.Printf("sessions: %d", len(sessions))
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := c.Value
	session, exists := sessions[token]
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if session.isExprired() {
		delete(sessions, token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = json.NewEncoder(w).Encode(session)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s Session) isExprired() bool {
	return s.Expiry.Before(time.Now())
}

func main() {
	_data := flag.String("data", "users.json", "user data file")
	_host := flag.String("host", "localhost", "host name")
	_port := flag.Int("port", 16000, "port number")
	flag.Parse()

	bytes, err := os.ReadFile(*_data)
	if err != nil {
		fmt.Println("failed to read user data file")
		os.Exit(1)
	}
	json.Unmarshal(bytes, &users)
	fmt.Printf("%d users loaded\n", len(users))

	http.HandleFunc("/login", Login)
	http.HandleFunc("/logout", Logout)
	http.HandleFunc("/refresh", Refresh)
	http.HandleFunc("/status/", Status)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *_host, *_port), nil))
}
