# aisec

AISec is a lightweight Go service for user registration and token-based authentication.

## Features

- Add user and generate API token
- Update user details
- Soft-delete user (`expired = true`)
- Authenticate user by username + token

## Requirements

- Go 1.22+
- PostgreSQL

## Environment

Copy `.env.example` to `.env` and update values as needed.

```bash
cp .env.example .env
```

Example values:

```env
DBHOST=db
DBPORT=5432
DBUSER=cfgmgr
DBPASSWORD=change-me
DBNAME=cfgmgr
SSLMode=disable
SECURITY_LISTENING_PORT=9097
```

## Database

Create table:

```sql
CREATE TABLE IF NOT EXISTS users (
  username TEXT PRIMARY KEY,
  password TEXT NOT NULL,
  api_token TEXT NOT NULL UNIQUE,
  expired BOOLEAN NOT NULL DEFAULT FALSE
);
```

## Run Locally

```bash
go mod tidy
go run .
```

## Run with Docker Compose

```bash
docker compose up -d --no-deps --force-recreate aisec
```

## API Endpoints

Base URL: `http://localhost:9097`

### Add User

- `POST /add-user`

Request:

```json
{
  "username": "alice",
  "password": "alice123"
}
```

Success response:

```json
{
  "message": "user added to db",
  "user": "alice",
  "token": "<generated-token>"
}
```

Duplicate user response:

- Status: `409 Conflict`

```json
{
  "message": "user already exists",
  "user": "alice",
  "token": ""
}
```

### Update User

- `PUT /update-user`

Request:

```json
{
  "username": "alice",
  "password": "new-password"
}
```

### Delete User

- `DELETE /delete-user?username=alice`

### Authenticate User

- `POST /authenticate-user`

Request:

```json
{
  "username": "alice",
  "api_token": "<token>"
}
```

Success response:

```json
{
  "message": "user authenticated",
  "user": "alice",
  "authenticated": true
}
```
