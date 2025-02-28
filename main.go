package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
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

func run() {
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

func initTeams() {
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

	reader := bufio.NewReader(os.Stdin)

	lib.Log.Query("Mathematically create teams? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	var inputs [][]string = make([][]string, 0)

	if response == "y" {
		lib.Log.Query("Enter number of teams: ")
		response, _ = reader.ReadString('\n')
		response = strings.TrimSpace(response)

		var numTeams int
		if _, err := fmt.Sscanf(response, "%d", &numTeams); err != nil {
			lib.Log.Error("Invalid number of teams")
			return
		}

		lib.Log.Query("Enter starting IPv4: ")
		response, _ = reader.ReadString('\n')
		ipv4 := strings.TrimSpace(response)

		for i := range numTeams {
			// Parse IPv4
			var octets []int = make([]int, 4)
			if _, err := fmt.Sscanf(ipv4, "%d.%d.%d.%d", &octets[0], &octets[1], &octets[2], &octets[3]); err != nil {
				lib.Log.Error("Invalid IPv4 address")
				return
			}

			// Increment last octet
			octets[3]++

			// Check for overflow
			if octets[3] > 255 {
				lib.Log.Error("IPv4 address overflow")
				return
			}

			// Reconstruct IPv4
			ipv4 = fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], octets[3])

			inputs = append(inputs, []string{fmt.Sprintf("Team %d", i+1), ipv4})
		}

		// Confirm
		for _, input := range inputs {
			lib.Log.Status(fmt.Sprintf("Team Name: %s, IPv4: %s", input[0], input[1]))
		}

		lib.Log.Query("Continue? (y/n): ")
		response, _ = reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" {
			lib.Log.Status("Aborted")
			return
		}
	} else {
		var teamNameRegex, teamIPRegex *regexp.Regexp = regexp.MustCompile("^[a-zA-Z0-9_\\- ]+$"), regexp.MustCompile("^(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$")

		for {
			var name, ipv4 string

			lib.Log.Query("Enter Team Name: ")
			name, _ = reader.ReadString('\n')
			name = strings.TrimSpace(name)

			lib.Log.Query("Enter team IPv4: ")
			ipv4, _ = reader.ReadString('\n')
			ipv4 = strings.TrimSpace(ipv4)

			if name == "" || ipv4 == "" {
				lib.Log.Error("Name and IPv4 cannot be empty")
				continue
			}

			// Validate name
			if !teamNameRegex.MatchString(name) {
				lib.Log.Error("Invalid team name")
				continue
			}

			// Validate IP
			if !teamIPRegex.MatchString(ipv4) {
				lib.Log.Error("Invalid IPv4 address")
				continue
			}

			// Check for duplicates
			for _, input := range inputs {
				if input[0] == name {
					lib.Log.Error("This team name is already in use")
					continue
				}

				if input[1] == ipv4 {
					lib.Log.Error("This IPv4 address is already in use")
					continue
				}
			}

			// Check for duplicates in environment
			if env.TeamByName(name) != nil {
				lib.Log.Error("This team name is already in use")
				continue
			}

			// Ping IP
			if lib.PingHost(ipv4) {
				lib.Log.Error("This IPv4 address is already in use")
				continue
			}

			inputs = append(inputs, []string{name, ipv4})

			var keepGoing bool
			for {
				lib.Log.Query("Continue? (y/n): ")
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response == "y" {
					keepGoing = true
					break
				} else if response == "n" {
					keepGoing = false
					break
				}
			}

			if !keepGoing {
				break
			}
		}
	}

	env.EfficientBulkCreate(inputs, 5)

	env.Print()
}

func purge() {
	if err := lib.InitEnv(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error initializing environment: %s", err))
		return
	} else {
		lib.Log.Status("Environment initialized")
	}

	proxmox, err := lib.InitProxmox()

	if err != nil {
		lib.Log.Error(fmt.Sprintf("Error initializing Proxmox: %s", err))
		return
	}

	lib.Log.Query("Are you sure you want to purge all King of the Hill instances? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" {
		lib.Log.Status("Purge aborted")
		return
	}

	lib.Log.Important("Removing SSH Keys")
	if err := os.Remove(lib.Config.SSH.PrivateKeyPath); err != nil {
		lib.Log.Error(fmt.Sprintf("Error removing private key: %s", err))
	} else {
		lib.Log.Status("Private key removed")
	}

	if err := os.Remove(lib.Config.SSH.PublicKeyPath); err != nil {
		lib.Log.Error(fmt.Sprintf("Error removing public key: %s", err))
	} else {
		lib.Log.Status("Public key removed")
	}

	lib.Log.Important("Removing database")
	if err := os.Remove(lib.Config.Database.File); err != nil {
		lib.Log.Error(fmt.Sprintf("Error removing database: %s", err))
	} else {
		lib.Log.Status("Database removed")
	}

	lib.Log.Important("Removing Proxmox containers")

	if containers, err := proxmox.RelevantContainers(); err != nil {
		lib.Log.Error(fmt.Sprintf("Error getting containers: %s", err))
	} else {
		for _, container := range containers {
			if err := proxmox.StopContainer(nil, int(container.VMID)); err != nil {
				lib.Log.Error(fmt.Sprintf("Error stopping container %d: %s", container.VMID, err))
			} else {
				lib.Log.Status(fmt.Sprintf("Container %d stopped", container.VMID))
			}

			if err := proxmox.DeleteContainer(nil, int(container.VMID)); err != nil {
				lib.Log.Error(fmt.Sprintf("Error deleting container %d: %s", container.VMID, err))
			} else {
				lib.Log.Status(fmt.Sprintf("Container %d deleted", container.VMID))
			}
		}
	}

	lib.Log.Success("Purge complete")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./koth <mode>\n\tuse 'modes' to see available modes")
		return
	}

	switch os.Args[1] {
	case "run":
		run()
	case "init":
		initTeams()
	case "purge":
		purge()
	default:
		fmt.Println("Available modes:")
		fmt.Println("\trun - Run the King of the Hill environment normally")
		fmt.Println("\tinit - Manually create teams through the CLI")
		fmt.Println("\tpurge - Destroy any and all king of the hill instances in Proxmox, wipe the database, remove keys.\n\t\tWill only remove proxmox containers with the name starting with env.CONTAINER_HOSTNAME_PREFIX")
	}
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
