package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type addUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type addUserResponse struct {
	Message string `json:"message"`
	User    string `json:"user"`
	Token   string `json:"token"`
}

type updateUserResponse struct {
	Message string `json:"message"`
	User    string `json:"user"`
}

type deleteUserResponse struct {
	Message string `json:"message"`
	User    string `json:"user"`
}

type authUserResponse struct {
	Message       string `json:"message"`
	User          string `json:"user"`
	Authenticated bool   `json:"authenticated"`
}

func readUserRequest(r *http.Request) (addUserRequest, error) {
	var req addUserRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, err
		}
	}

	if req.Username == "" {
		req.Username = r.URL.Query().Get("username")
	}
	if req.Password == "" {
		req.Password = r.URL.Query().Get("password")
	}

	return req, nil
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed; use POST", http.StatusMethodNotAllowed)
		return
	}

	req, err := readUserRequest(r)
	if err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user := User{
		Username: req.Username,
		Password: req.Password,
	}
	if err := user.AddUser(); err != nil {
		if errors.Is(err, ErrUserExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(addUserResponse{
				Message: "user already exists",
				User:    user.Username,
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(addUserResponse{
		Message: "user added to db",
		User:    user.Username,
		Token:   user.Apitoken,
	})
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	addHandler(w, r)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	req, err := readUserRequest(r)
	if err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	user := User{
		Username: req.Username,
		Password: req.Password,
		Apitoken: r.URL.Query().Get("api_token"),
	}
	if user.Apitoken == "" {
		user.Apitoken = r.Header.Get("X-API-Token")
	}

	if err := user.UpdateUser(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updateUserResponse{
		Message: "user updated",
		User:    user.Username,
	})
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateHandler(w, r)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		var req addUserRequest
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}
		username = req.Username
	}

	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	user := User{Username: username}
	if err := user.DeleteUser(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(deleteUserResponse{
		Message: "user deleted",
		User:    user.Username,
	})
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	deleteHandler(w, r)
}

func authenticateHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	apiToken := r.URL.Query().Get("api_token")

	if username == "" || apiToken == "" {
		var payload struct {
			Username string `json:"username"`
			Apitoken string `json:"api_token"`
		}
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}
		if username == "" {
			username = payload.Username
		}
		if apiToken == "" {
			apiToken = payload.Apitoken
		}
	}

	if username == "" || apiToken == "" {
		http.Error(w, "username and api token are required", http.StatusBadRequest)
		return
	}

	user := User{
		Username: username,
		Apitoken: apiToken,
	}
	authed, err := user.AuthenticateUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	message := "user authenticated"
	if !authed {
		status = http.StatusUnauthorized
		message = "authentication failed"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(authUserResponse{
		Message:       message,
		User:          user.Username,
		Authenticated: authed,
	})
}

func AuthenticateHandler(w http.ResponseWriter, r *http.Request) {
	authenticateHandler(w, r)
}
