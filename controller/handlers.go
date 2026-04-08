package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type addUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
	Expired     *bool  `json:"expired"`
}

type updateUserRequestAliases struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"newPassword"`
}

type addUserResponse struct {
	Message string `json:"message"`
	User    string `json:"user"`
	Token   string `json:"token"`
}

type updateUserResponse struct {
	Message string `json:"message"`
	User    string `json:"user"`
	Token   string `json:"token"`
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

type verifyTokenResponse struct {
	Message string `json:"message"`
	Valid   bool   `json:"valid"`
	Expired bool   `json:"expired"`
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

func readUpdateUserRequest(r *http.Request) (updateUserRequest, error) {
	var req updateUserRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return req, err
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				return req, err
			}
			var aliasReq updateUserRequestAliases
			if err := json.Unmarshal(body, &aliasReq); err == nil {
				if req.Password == "" {
					req.Password = aliasReq.CurrentPassword
				}
				if req.NewPassword == "" {
					req.NewPassword = aliasReq.NewPassword
				}
			}
		}
	}

	if req.Username == "" {
		req.Username = r.URL.Query().Get("username")
	}
	if req.Password == "" {
		req.Password = r.URL.Query().Get("password")
	}
	if req.Password == "" {
		req.Password = r.URL.Query().Get("current_password")
	}
	if req.NewPassword == "" {
		req.NewPassword = r.URL.Query().Get("new_password")
	}
	if req.NewPassword == "" {
		req.NewPassword = r.URL.Query().Get("newPassword")
	}
	if req.Expired == nil {
		if expiredText := r.URL.Query().Get("expired"); expiredText != "" {
			expired, err := strconv.ParseBool(expiredText)
			if err != nil {
				return req, err
			}
			req.Expired = &expired
		}
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
	req, err := readUpdateUserRequest(r)
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
	}

	if err := user.UpdateUser(req.Password, req.NewPassword, req.Expired); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrInvalidCurrentPassword) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updateUserResponse{
		Message: "user updated",
		User:    user.Username,
		Token:   user.Apitoken,
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

func readTokenFromRequest(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token != "" {
			return token, nil
		}
	}

	if apiToken := strings.TrimSpace(r.URL.Query().Get("api_token")); apiToken != "" {
		return apiToken, nil
	}

	var payload struct {
		Apitoken string `json:"api_token"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return "", err
		}
	}

	return strings.TrimSpace(payload.Apitoken), nil
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	token, err := readTokenFromRequest(r)
	if err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if token == "" {
		http.Error(w, "api token is required", http.StatusBadRequest)
		return
	}

	user := User{Apitoken: token}
	exists, expired, err := user.VerifyToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	message := "token valid"
	valid := true
	if !exists {
		status = http.StatusUnauthorized
		message = "token not found"
		valid = false
		expired = false
	} else if expired {
		status = http.StatusUnauthorized
		message = "token expired"
		valid = false
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(verifyTokenResponse{
		Message: message,
		Valid:   valid,
		Expired: expired,
	})
}

func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	verifyHandler(w, r)
}
