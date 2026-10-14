# Chirpy

Chirpy is a RESTful backend API for a Twitter-like social platform, built with **Go and PostgreSQL**.

The application allows users to register, authenticate, create and manage short posts (chirps), and retrieve posts from other users. It implements secure password hashing, JWT-based authentication, refresh token management, and a simulated payment webhook for premium membership upgrades.

## Features

### User Management

- User registration with email and password
- Secure password hashing using Argon2id
- User authentication and login
- Update account email and password
- Premium membership status (Chirpy Red)

### Authentication & Security

- JWT access tokens using HS256
- Short-lived access tokens (1-hour expiration)
- Database-backed refresh tokens (60-day expiration)
- Refresh token revocation
- Protected endpoints requiring valid authentication
- Resource ownership verification for deleting chirps
- API key authentication for webhook requests

### Chirp Management

- Create chirps with a maximum length of 140 characters
- Retrieve all chirps
- Retrieve individual chirps by ID
- Filter chirps by author
- Sort chirps by creation date (ascending or descending)
- Delete chirps with ownership verification
- Automatic profanity filtering

### Additional Features

- RESTful API endpoints returning JSON
- PostgreSQL database integration
- Database migrations with Goose
- Type-safe SQL queries generated with sqlc
- Webhook integration for premium membership upgrades
- HTTP request handling using Go's standard library
- Application configuration using environment variables

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Backend development |
| net/http | HTTP server and routing |
| PostgreSQL | Relational database |
| sqlc | Type-safe Go code generation from SQL |
| Goose | Database migrations |
| Argon2id | Password hashing |
| golang-jwt/jwt/v5 | JWT generation and validation |
| google/uuid | UUID generation and handling |
| godotenv | Environment variable management |

## Getting Started

### Prerequisites

Before running Chirpy, make sure you have the following installed:

- Go (1.22 or later)
- PostgreSQL
- Goose
- sqlc

### 1. Clone the Repository

```bash
git clone https://github.com/ha0ch/chirpy.git
cd chirpy
```

### 2. Install Dependencies

Download the required Go dependencies:

```bash
go mod download
```

### 3. Configure PostgreSQL

Start your PostgreSQL server and create a database named `chirpy`.

For example:

```bash
createdb chirpy
```

Make sure your PostgreSQL server is running before starting the application.

### 4. Configure Environment Variables

Create a `.env` file in the project root:

```env
DB_URL=postgres://YOUR_USERNAME@localhost:5432/chirpy
PLATFORM=dev
JWT_SECRET=your_random_secret
POLKA_KEY=your_polka_api_key
```

**Environment variables:**

| Variable | Description |
|---|---|
| `DB_URL` | PostgreSQL connection string |
| `PLATFORM` | Application environment (`dev` enables the admin reset endpoint) |
| `JWT_SECRET` | Secret used to sign and validate JWTs |
| `POLKA_KEY` | API key used to authenticate Polka webhook requests |

Generate a secure JWT secret with:

```bash
openssl rand -base64 64
```

**Security:** Never commit your `.env` file or actual secrets to GitHub.

### 5. Run Database Migrations

Apply the database migrations using Goose:

```bash
goose -dir sql/schema postgres "postgres://YOUR_USERNAME@localhost:5432/chirpy" up
```

Replace `YOUR_USERNAME` with your PostgreSQL username.

### 6. Generate Database Code

Generate Go code from the SQL queries:

```bash
sqlc generate
```

### 7. Build and Run

Build the application:

```bash
go build -o out
```

Start the server:

```bash
./out
```

Alternatively, run it directly:

```bash
go run .
```

The API runs at:

```
http://localhost:8080
```

## API Endpoints

### Users

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| POST | `/api/users` | Register a new user | None |
| POST | `/api/login` | Authenticate and receive access and refresh tokens | None |
| PUT | `/api/users` | Update the authenticated user's email and password | JWT |

### Chirps

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| POST | `/api/chirps` | Create a new chirp | JWT |
| GET | `/api/chirps` | Retrieve all chirps | None |
| GET | `/api/chirps/{chirpID}` | Retrieve a chirp by ID | None |
| DELETE | `/api/chirps/{chirpID}` | Delete a chirp owned by the authenticated user | JWT |

**Optional query parameters for `GET /api/chirps`:**

| Parameter | Description | Example |
|---|---|---|
| `author_id` | Filter chirps by author UUID | `?author_id=<uuid>` |
| `sort` | Sort by creation time (`asc` or `desc`) | `?sort=desc` |

Parameters can be combined:

```http
GET /api/chirps?author_id=<uuid>&sort=desc
```

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/refresh` | Generate a new JWT using a valid refresh token |
| POST | `/api/revoke` | Revoke a refresh token |

### Webhooks

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| POST | `/api/polka/webhooks` | Process premium membership upgrades | API Key |

### Administration

| Method | Endpoint | Description |
|---|---|---|
| POST | `/admin/reset` | Reset application data in development mode |

The reset endpoint is restricted to development mode.

## API Usage Examples

### 1. Register a User

**Request**

```http
POST /api/users
Content-Type: application/json
```

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response — 201 Created**

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "created_at": "2026-10-12T12:00:00Z",
  "updated_at": "2026-10-12T12:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

Passwords are securely hashed before being stored in the database.

### 2. Log In

**Request**

```http
POST /api/login
Content-Type: application/json
```

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response — 200 OK**

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "created_at": "2026-10-12T12:00:00Z",
  "updated_at": "2026-10-12T12:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false,
  "token": "<JWT_ACCESS_TOKEN>",
  "refresh_token": "<REFRESH_TOKEN>"
}
```

The access token expires after one hour, while the refresh token remains valid for 60 days unless revoked.

### 3. Create a Chirp

This endpoint requires a valid JWT.

**Request**

```http
POST /api/chirps
Authorization: Bearer <JWT_ACCESS_TOKEN>
Content-Type: application/json
```

```json
{
  "body": "Hello, Chirpy!"
}
```

**Response — 201 Created**

```json
{
  "id": "223e4567-e89b-12d3-a456-426614174000",
  "created_at": "2026-10-12T12:05:00Z",
  "updated_at": "2026-10-12T12:05:00Z",
  "body": "Hello, Chirpy!",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

The server identifies the author using the validated JWT rather than relying on a user ID provided in the request body.

### 4. Retrieve and Filter Chirps

Retrieve all chirps:

```http
GET /api/chirps
```

Retrieve chirps from a specific user:

```http
GET /api/chirps?author_id=<uuid>
```

Retrieve chirps sorted from newest to oldest:

```http
GET /api/chirps?sort=desc
```

Combine filtering and sorting:

```http
GET /api/chirps?author_id=<uuid>&sort=desc
```

### 5. Refresh an Access Token

**Request**

```http
POST /api/refresh
Authorization: Bearer <REFRESH_TOKEN>
```

**Response — 200 OK**

```json
{
  "token": "<NEW_JWT_ACCESS_TOKEN>"
}
```

The server verifies the refresh token against the database, checking that it hasn't expired or been revoked.

### 6. Revoke a Refresh Token

**Request**

```http
POST /api/revoke
Authorization: Bearer <REFRESH_TOKEN>
```

**Response — 204 No Content**

After revocation, the refresh token can no longer be used to generate new access tokens.

### 7. Delete a Chirp

Only the original author can delete a chirp.

**Request**

```http
DELETE /api/chirps/<chirpID>
Authorization: Bearer <JWT_ACCESS_TOKEN>
```

**Possible responses:**

| Status | Meaning |
|---|---|
| 204 No Content | Chirp successfully deleted |
| 401 Unauthorized | Missing or invalid JWT |
| 403 Forbidden | User does not own the chirp |
| 404 Not Found | Chirp does not exist |

### 8. Upgrade a User to Chirpy Red

Premium membership upgrades are handled through a simulated Polka payment webhook.

**Request**

```http
POST /api/polka/webhooks
Authorization: ApiKey <POLKA_KEY>
Content-Type: application/json
```

```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000"
  }
}
```

**Response — 204 No Content**

The webhook verifies the API key before updating the user's premium membership status.

## Database

Chirpy uses PostgreSQL to persist application data.

The database includes three main tables:

### Users

Stores account information, password hashes, timestamps, and Chirpy Red membership status.

### Chirps

Stores posts and associates them with their authors using foreign keys.

### Refresh Tokens

Stores refresh tokens, their associated users, expiration timestamps, and revocation timestamps.

Foreign key constraints with `ON DELETE CASCADE` ensure that deleting a user also removes their associated chirps and refresh tokens.

### Database Migrations

Database schema changes are managed using Goose.

Apply pending migrations:

```bash
goose -dir sql/schema postgres "postgres://YOUR_USERNAME@localhost:5432/chirpy" up
```

Check migration status:

```bash
goose -dir sql/schema postgres "postgres://YOUR_USERNAME@localhost:5432/chirpy" status
```

Roll back the latest migration:

```bash
goose -dir sql/schema postgres "postgres://YOUR_USERNAME@localhost:5432/chirpy" down
```

### SQL Code Generation

Chirpy uses sqlc to generate type-safe Go methods directly from SQL queries.

This approach allows SQL to be written explicitly while using generated Go structs and methods for database operations.

Regenerate database code after modifying queries or schemas:

```bash
sqlc generate
```

## Testing

Run all Go tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

The project includes unit tests for authentication functionality, including JWT creation and validation.

Additional endpoint functionality can be tested using HTTP clients such as curl or Postman.

## Security Considerations

Chirpy implements several security practices:

- **Password hashing:** Passwords are hashed using Argon2id before storage.
- **JWT authentication:** Protected endpoints require signed access tokens.
- **Token expiration:** Access and refresh tokens have limited lifetimes.
- **Token revocation:** Refresh tokens can be invalidated through the database.
- **Authorization:** Users can only modify or delete resources they own.
- **Webhook authentication:** Incoming Polka webhook requests require a valid API key.
- **Environment variables:** Sensitive configuration is kept outside the source code.

Chirpy is an educational backend project and is not intended to be deployed as a production-ready social platform without additional security hardening.

## What I Learned

Building Chirpy provided hands-on experience with:

- Designing and implementing RESTful APIs in Go
- Working with HTTP request handlers and middleware
- Using PostgreSQL and relational database constraints
- Writing SQL queries and managing database migrations
- Generating type-safe Go database code with sqlc
- Implementing password hashing and authentication
- Creating, validating, and revoking authentication tokens
- Implementing authorization and resource ownership checks
- Handling third-party webhooks
- Managing application configuration through environment variables

## Acknowledgments

Chirpy was developed as part of the [Boot.dev](https://www.boot.dev/) backend development curriculum, with the goal of strengthening practical skills in Go, SQL, HTTP, and backend application development.

The project may be independently expanded with additional
features and functionality.

## License

Copyright © 2026 Hao Chen. All rights reserved.

No license is granted for copying, modifying, or distributing
this project's source code.
