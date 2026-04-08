package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type PostgresqlConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
	SSLMode  string
}

func (p PostgresqlConfig) String() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.Database,
		p.SSLMode,
	)
}

func dbInitialize() (*pgx.Conn, error) {
	cfg := PostgresqlConfig{
		Host:     os.Getenv("DBHOST"),
		Port:     os.Getenv("DBPORT"),
		User:     os.Getenv("DBUSER"),
		Password: os.Getenv("DBPASSWORD"),
		Database: os.Getenv("DBNAME"),
		SSLMode:  os.Getenv("SSLMode"),
	}
	conn, err := pgx.Connect(context.Background(), cfg.String())
	if err != nil {
		return nil, err
	}
	return conn, nil
}
