package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
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
	expireTime time.Duration // Sessions expire after this duration
	sessionDir string        // Directory to store active sessions
	users      []User
	sessions   = map[string]Session{}
)

func createSession(token string, user string, expires time.Time) (*Session, error) {
	err := os.WriteFile(sessionDir+"/"+token, []byte(fmt.Sprintf("%s %s", user, expires.Format(time.UnixDate))), 0644)
	if err != nil {
		return nil, err
	}
	s := Session{user, expires}
	sessions[token] = s
	return &s, nil
}

func deleteSession(token string) {
	os.Remove(sessionDir + "/" + token)
	delete(sessions, token)
}

func (s Session) isExprired() bool {
	return s.Expiry.Before(time.Now())
}

func Login(w http.ResponseWriter, r *http.Request) {
	// TODO: check if already logged in
	// c, err := r.Cookie("session_token")
	// fmt.Println(r.Method)
	r.ParseForm()
	if !r.Form.Has("name") || !r.Form.Has("password") {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	i := slices.IndexFunc(users, func(u User) bool {
		return r.Form.Get("name") == u.Name && r.Form.Get("password") == u.Password
	})
	if i == -1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user := users[i]
	token := uuid.NewString()
	expiresAt := time.Now().Add(expireTime)
	createSession(token, user.Name, expiresAt)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expiresAt,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
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
	deleteSession(token)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
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
		deleteSession(token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	deleteSession(token)
	newToken := uuid.NewString()
	expiresAt := time.Now().Add(expireTime)
	createSession(newToken, session.User, expiresAt)
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

func List(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handler:list")
	fmt.Printf("sessions: %d", len(sessions))
	for k, v := range sessions {
		fmt.Println(k, v)
	}
}

func main() {
	_host := flag.String("host", "localhost", "host name")
	_port := flag.Int("port", 16000, "port number")
	_data := flag.String("data", "users.json", "user data file")
	_clean := flag.Bool("clean", false, "clear existing sessions")
	_expire := flag.Int("expire", 60, "expire time in minutes")
	_session_dir := flag.String("session-dir", "/tmp/user-session", "session storage directory")
	flag.Parse()

	expireTime = time.Duration(*_expire) * time.Minute

	bytes, err := os.ReadFile(*_data)
	if err != nil {
		log.Fatal("failed to read user data file [" + *_data + "]")
	}
	json.Unmarshal(bytes, &users)
	fmt.Printf("%d users loaded\n", len(users))
	for _, u := range users {
		fmt.Println("  󰘍 ", u.Name)
	}

	sessionDir = *_session_dir
	_, err = os.Stat(sessionDir)
	if os.IsNotExist(err) {
		if err := os.Mkdir(sessionDir, 0750); err != nil {
			log.Fatal(err)
		}
	} else {
		entries, _ := os.ReadDir(sessionDir)
		if *_clean {
			for _, e := range entries {
				os.Remove(sessionDir + "/" + e.Name())
			}
		} else {
			for _, e := range entries {
				token := e.Name()
				path := sessionDir + "/" + token
				file, _ := os.ReadFile(path)
				line := string(file)
				i := strings.Index(line, " ")
				username := line[:i]
				expireAt, _ := time.Parse(time.UnixDate, line[i+1:])
				i = slices.IndexFunc(users, func(u User) bool {
					return u.Name == username
				})
				if i == -1 {
					fmt.Printf("no user found for stored session %s\n", token)
					os.Remove(path)
					continue
				}
				if expireAt.Before(time.Now()) {
					fmt.Println("session timeout")
					os.Remove(path)
					continue
				}
				sessions[token] = Session{username, expireAt}
			}
		}
		fmt.Printf("%d sessions loaded\n", len(sessions))
		for k, v := range sessions {
			fmt.Println("  󰘍 ", k, v)
		}
	}

	http.HandleFunc("/session/login", Login)
	http.HandleFunc("/session/logout", Logout)
	http.HandleFunc("/session/refresh", Refresh)
	http.HandleFunc("/session/status/", Status)
	http.HandleFunc("/session/list/", List)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *_host, *_port), nil))
}
