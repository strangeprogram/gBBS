package web

import (
	"encoding/json"
	"fmt"
	"gbbs/internal/messageboard"
	"gbbs/internal/user"
	"net/http"
)

func Serve(port int, webRoot string, userManager *user.Manager, messageBoard *messageboard.MessageBoard) error {
	http.Handle("/", http.FileServer(http.Dir(webRoot)))
	http.HandleFunc("/api/login", loginHandler(userManager))
	http.HandleFunc("/api/register", registerHandler(userManager))
	http.HandleFunc("/api/messages", messagesHandler(messageBoard))

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func loginHandler(userManager *user.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		authenticated, err := userManager.Authenticate(creds.Username, creds.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !authenticated {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
	}
}

func registerHandler(userManager *user.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := userManager.CreateUser(creds.Username, creds.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
	}
}

func messagesHandler(messageBoard *messageboard.MessageBoard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			messages, err := messageBoard.GetMessages()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(messages)
		case http.MethodPost:
			var msg struct {
				Username string `json:"username"`
				Message  string `json:"message"`
			}
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := messageBoard.PostMessage(msg.Username, msg.Message); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"message": "Message posted successfully"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
