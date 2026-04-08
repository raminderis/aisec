package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrUserExists = errors.New("user already exists")

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
	u.Expired = false

	u.GenerateToken()

	conn, err := dbInitialize()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(
		context.Background(),
		"INSERT INTO users (username, password, api_token, expired) VALUES ($1, $2, $3, $4)",
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

func (u *User) UpdateUser() error {
	u.Username = strings.TrimSpace(u.Username)
	u.Password = strings.TrimSpace(u.Password)
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
		"UPDATE users SET password = $2, api_token = $3, expired = $4 WHERE username = $1 AND (password IS DISTINCT FROM $2 OR api_token IS DISTINCT FROM $3 OR expired IS DISTINCT FROM $4)",
		u.Username,
		u.Password,
		u.Apitoken,
		u.Expired,
	)
	if err != nil {
		return err
	}

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
		"SELECT username, password, api_token, expired FROM users WHERE username = $1 AND api_token = $2 LIMIT 1",
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
