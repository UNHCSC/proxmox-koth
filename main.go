package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"koth.cyber.cs.unh.edu/database"
	"koth.cyber.cs.unh.edu/environment"
	"koth.cyber.cs.unh.edu/lib"
)

type Token struct {
	Token   string
	Expires time.Time
}

func (t *Token) Expired() bool {
	return t.Expires.Before(time.Now())
}

var tokens []*Token = make([]*Token, 0)

func RandomString(length int) string {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)

	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", bytes)
}

func NewToken() *Token {
	var token *Token = &Token{
		Token:   RandomString(48),
		Expires: time.Now().Add(time.Hour),
	}

	tokens = append(tokens, token)

	return token
}

func TokenFor(token string) *Token {
	for _, t := range tokens {
		if t.Token == token {
			return t
		}
	}

	return nil
}

func DeleteToken(token string) {
	for i, t := range tokens {
		if t.Token == token {
			tokens = append(tokens[:i], tokens[i+1:]...)
			return
		}
	}
}

func CleanTokens() {
	var newTokens []*Token = make([]*Token, 0)
	for _, t := range tokens {
		if !t.Expired() {
			newTokens = append(newTokens, t)
		}
	}

	tokens = newTokens
}

func withCors(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
}

func withAuth(w http.ResponseWriter, r *http.Request) bool {

	token, err := r.Cookie("token")

	if err != nil || token == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	if t := TokenFor(token.Value); t == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	} else {
		if t.Expired() {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		t.Expires = time.Now().Add(time.Hour)
	}

	return true
}

func main() {
	if err := lib.InitEnv(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error initializing environment: %s", err))
		return
	} else {
		lib.Log.Status("Environment initialized")
	}

	if err := lib.InitSSH(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error initializing SSH: %s", err))
		return
	} else {
		lib.Log.Status("SSH initialized")
	}

	if err := database.Connect(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error connecting to database: %s", err))
		return
	} else {
		lib.Log.Status("Database connected")
	}

	proxmox, err := lib.InitProxmox()

	if err != nil {
		lib.Log.Error(fmt.Sprintf("Error initializing Proxmox: %s", err))
		return
	}

	var env *environment.Environment = environment.NewEnvironment(proxmox)

	if err := env.PullFromDatabase(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error pulling from database: %s", err))
		return
	} else {
		lib.Log.Status("Environment pulled from database")
	}

	env.Print()

	// for i := 1; i <= 5; i++ {
	// 	go func() {
	// 		if _, err := env.CreateContainer(fmt.Sprintf("Team %d", i), fmt.Sprintf("192.168.7.%d", 238+i), true); err != nil {
	// 			lib.Log.Error(err.Error())
	// 		} else {
	// 			lib.Log.Status(fmt.Sprintf("[%s][%s]: Team Initialized", fmt.Sprintf("Team %d", i), fmt.Sprintf("192.168.7.%d", 238+i)))
	// 		}
	// 	}()

	// 	time.Sleep(15 * time.Second)
	// }

	envUpdateChannel := env.InitAutoUpdate()

	// Serve static files w/ CORS
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.ServeFile(w, r, "./public"+r.URL.Path)
	})

	http.HandleFunc("/api/checkLogin", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if !withAuth(w, r) {
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if obj.Username != lib.Config.WebServer.Username || obj.Password != lib.Config.WebServer.Password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    NewToken().Token,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if !withAuth(w, r) {
			return
		}

		if r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		// Get cookie
		token, err := r.Cookie("token")

		if err != nil || token == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if t := TokenFor(token.Value); t == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			DeleteToken(token.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/public/summary.json", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		json, err := env.JSON()

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	})

	go func() {
		for {
			CleanTokens()
			time.Sleep(time.Minute)
		}
	}()

	lib.Log.Status(fmt.Sprintf("Server started on port %d", lib.Config.WebServer.Port))
	var at string = fmt.Sprintf("%s:%d", lib.Config.WebServer.Host, lib.Config.WebServer.Port)

	if lib.Config.WebServer.TlsDir != "" {
		http.ListenAndServeTLS(at, lib.Config.WebServer.TlsDir+"/fullchain.pem", lib.Config.WebServer.TlsDir+"/privkey.pem", nil)
	} else {
		http.ListenAndServe(at, nil)
	}

	// Cleanup
	envUpdateChannel <- true
}

// func main() {
// 	lib.InitEnv()
// 	proxmox, err := lib.InitProxmox()

// 	if err != nil {
// 		lib.Log.Error(fmt.Sprintf("Error initializing Proxmox: %s", err))
// 		return
// 	}

// 	var ids []int = []int{105, 129, 130, 131, 132}

// 	for _, id := range ids {
// 		if err := proxmox.StopContainer(nil, id); err != nil {
// 			lib.Log.Error(fmt.Sprintf("Error stopping container %d: %s", id, err))
// 		} else {
// 			lib.Log.Status(fmt.Sprintf("Container %d stopped", id))
// 		}

// 		if err := proxmox.DeleteContainer(nil, id); err != nil {
// 			lib.Log.Error(fmt.Sprintf("Error deleting container %d: %s", id, err))
// 		} else {
// 			lib.Log.Status(fmt.Sprintf("Container %d deleted", id))
// 		}
// 	}
// }
