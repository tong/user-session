package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// const sessionExpireTime = 120 * time.Second
const sessionExpireTime = 120 * time.Minute

var users = map[string]string{
	"tong":   "test",
	"kilmou": "test",
}

var sessions = map[string]Session{}

type Session struct {
	Username string    `json:"username"`
	Expiry   time.Time `json:"expiry"`
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

func (s Session) isExprired() bool {
	return s.Expiry.Before(time.Now())
}

func Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("signin...")
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
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
	expiresAt := time.Now().Add(sessionExpireTime)
	// sessions[sessionToken] = session{
	// 	username: creds.Username,
	// 	expiry:   expiresAt,
	// }
	sessions[sessionToken] = Session{creds.Username, expiresAt}
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
	expiresAt := time.Now().Add(sessionExpireTime)
	// sessions[newSessionToken] = session{
	// 	username: userSession.username,
	// 	expiry:   expiresAt,
	// }
	sessions[newSessionToken] = Session{userSession.Username, expiresAt}
	delete(sessions, sessionToken)
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(sessionExpireTime),
	})
}

func Status(w http.ResponseWriter, r *http.Request) {
	fmt.Println("status...")
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
	// w.Write([]byte(fmt.Sprintf("user: %s\nexpires: %s", userSession.username, userSession.expiry)))
	// w.Write([]byte(fmt.Sprintf("{\"user\": \"%s\", \"expires\": \"%s\"}", userSession.username, userSession.expiry)))
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(userSession)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func Register(w http.ResponseWriter, r *http.Request) {
	fmt.Println("regster...TODO")
}

func Unregister(w http.ResponseWriter, r *http.Request) {
	fmt.Println("unregster...TODO")
}

/*
func Test(w http.ResponseWriter, r *http.Request) {
	fmt.Println("test...", r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// data := TestData{Username: "king"}
	data := TestData{Username: "king"}
	// data := `{"username": "king"}`
	fmt.Println(data)
	// fmt.Println(data.username)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// w.WriteHeader(http.StatusOK)
}

type TestData struct {
	// Username string `json:"username"`
	Username string
}
*/

func main() {
	/*
		// data := `{"username": "king"}`
		data := TestData{"king"}
		// err := json.Unmarshal([]byte(data), &obj)
		b, err := json.Marshal(data)
		fmt.Println(err)
		fmt.Println(b)

		var obj TestData
		err = json.Unmarshal(b, &obj)
		fmt.Println(err)
		fmt.Println(obj)
	*/
	// http.HandleFunc("/headers", headers)
	// http.HandleFunc("/", Home)
	http.HandleFunc("/auth/login", Login)
	http.HandleFunc("/auth/logout", Logout)
	http.HandleFunc("/auth/refresh", Refresh)
	http.HandleFunc("/auth/status", Status)
	http.HandleFunc("/auth/register", Register)
	http.HandleFunc("/auth/unregister", Unregister)
	// http.HandleFunc("/auth/test", Test)
	http.ListenAndServe(":16000", nil)
}
