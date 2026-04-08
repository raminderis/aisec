package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidCurrentPassword = errors.New("current password is invalid")

type User struct {
	Username string
	Password string
	Apitoken string
	Expired  bool
}

func (u *User) AddUser() error {
	u.Username = strings.TrimSpace(u.Username)
	u.Password = strings.TrimSpace(u.Password)
	if u.Username == "" || u.Password == "" {
		return errors.New("username and password are required")
	}
	hashedPassword, err := hashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	u.Expired = false

	u.GenerateToken()

	conn, err := dbInitialize()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(
		context.Background(),
		"INSERT INTO users (username, password_hash, api_token, expired) VALUES ($1, $2, $3, $4)",
		u.Username,
		u.Password,
		u.Apitoken,
		u.Expired,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserExists
		}
		return err
	}

	return nil
}

func (u *User) UpdateUser(currentPassword string, newPassword string, expired *bool) error {
	u.Username = strings.TrimSpace(u.Username)
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if u.Username == "" {
		return errors.New("username is required")
	}
	if currentPassword == "" {
		return errors.New("password is required")
	}

	conn, err := dbInitialize()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	var currentPasswordStored string
	var currentToken string
	var currentExpired bool
	err = conn.QueryRow(
		context.Background(),
		"SELECT password_hash, api_token, expired FROM users WHERE username = $1 LIMIT 1",
		u.Username,
	).Scan(&currentPasswordStored, &currentToken, &currentExpired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	currentPasswordValid, err := passwordMatches(currentPasswordStored, currentPassword)
	if err != nil {
		return err
	}
	if !currentPasswordValid {
		return ErrInvalidCurrentPassword
	}

	updatedPassword := currentPasswordStored
	updatedToken := currentToken
	updatedExpired := currentExpired
	shouldUpdate := false

	if newPassword != "" {
		samePassword, err := passwordMatches(currentPasswordStored, newPassword)
		if err != nil {
			return err
		}
		if !samePassword {
			hashedNewPassword, err := hashPassword(newPassword)
			if err != nil {
				return err
			}
			updatedPassword = hashedNewPassword
			u.GenerateToken()
			updatedToken = u.Apitoken
			shouldUpdate = true
		}
	} else {
		u.Apitoken = currentToken
	}

	if expired != nil && *expired != currentExpired {
		updatedExpired = *expired
		shouldUpdate = true
	}

	if !shouldUpdate {
		u.Password = currentPasswordStored
		u.Expired = currentExpired
		return nil
	}

	_, err = conn.Exec(
		context.Background(),
		"UPDATE users SET password_hash = $2, api_token = $3, expired = $4 WHERE username = $1",
		u.Username,
		updatedPassword,
		updatedToken,
		updatedExpired,
	)
	if err != nil {
		return err
	}

	u.Password = updatedPassword
	u.Apitoken = updatedToken
	u.Expired = updatedExpired

	return nil
}

func (u *User) GenerateToken() {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		u.Apitoken = "token-pending"
		return
	}
	u.Apitoken = hex.EncodeToString(b)
}

func (u *User) DeleteUser() error {
	u.Username = strings.TrimSpace(u.Username)
	if u.Username == "" {
		return errors.New("username is required")
	}

	conn, err := dbInitialize()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(
		context.Background(),
		"UPDATE users SET expired = true WHERE username = $1",
		u.Username,
	)
	if err != nil {
		return err
	}

	u.Expired = true
	return nil
}

func (u *User) AuthenticateUser() (bool, error) {
	u.Username = strings.TrimSpace(u.Username)
	u.Apitoken = strings.TrimSpace(u.Apitoken)
	if u.Username == "" || u.Apitoken == "" {
		return false, errors.New("username and api token are required")
	}

	conn, err := dbInitialize()
	if err != nil {
		return false, err
	}
	defer conn.Close(context.Background())

	err = conn.QueryRow(
		context.Background(),
		"SELECT username, password_hash, api_token, expired FROM users WHERE username = $1 AND api_token = $2 LIMIT 1",
		u.Username,
		u.Apitoken,
	).Scan(&u.Username, &u.Password, &u.Apitoken, &u.Expired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if u.Expired {
		return false, nil
	}

	return true, nil
}

func (u *User) VerifyToken() (bool, bool, error) {
	u.Apitoken = strings.TrimSpace(u.Apitoken)
	if u.Apitoken == "" {
		return false, false, errors.New("api token is required")
	}

	conn, err := dbInitialize()
	if err != nil {
		return false, false, err
	}
	defer conn.Close(context.Background())

	err = conn.QueryRow(
		context.Background(),
		"SELECT expired FROM users WHERE api_token = $1 LIMIT 1",
		u.Apitoken,
	).Scan(&u.Expired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}

	return true, u.Expired, nil
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func passwordMatches(storedPassword string, providedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(providedPassword))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}

	return false, err
}
