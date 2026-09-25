## Local HTTPS (OAuth-friendly)

To run the backend over HTTPS (required by Facebook and recommended for OAuth callbacks):

1. Install mkcert (preferred) or ensure OpenSSL is available.
2. Run: `make run-https`

This will:

- Generate a trusted localhost certificate (via mkcert) or a self-signed one as fallback.
- Export TLS_ENABLED=1 and certificate paths for the server.
- Default OAuth callbacks to:
  - https://localhost:10001/auth/youtube/callback
  - https://localhost:10001/auth/facebook/callback

You can override with env vars `YOUTUBE_REDIRECT_URL` and `FACEBOOK_REDIRECT_URL`.

Angular admin app can be run with `ng serve --ssl --proxy-config proxy.conf.json` from the FE project.
# Go Project - API Documentation

## Table of Contents
- [Description](#description)
- [Requirements](#requirements)
- [API Endpoints](#api-endpoints)
- [Authentication](#authentication)
- [User Management](#user-management)
- [YouTube API Integration](#youtube-api-integration)
- [Setup](#setup)
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)
 - [Migrations](#migrations)
- [Real-time Share Status (SSE)](#real-time-share-status-sse)
- [Architecture (Detailed Doc)](./docs/ARCHITECTURE.md)

## Description
This project is a Go REST API with comprehensive user management and YouTube API integration. It provides endpoints for user authentication, video management, channel operations, and comment handling.

**Base URL:** `http://localhost:10001`

## Requirements
- Go 1.21+
- PostgreSQL 5.7+ (Primary database)
- MySQL 5.7 (Optional)
- MongoDB (Optional)
- Liquibase 4.3.5
- Docker 20.10.7
- Docker Compose 1.29.2
- Google Cloud Console account (for YouTube API)

## API Endpoints

### 🔑 Authentication

#### Register User
```bash
curl -X POST http://localhost:10001/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "user_name": "johndoe",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "response_code": "200",
  "response_message": "Success",
  "data": {
    "name": "John Doe",
    "user_name": "johndoe"
  }
}
```

#### Login User
```bash
curl -X POST http://localhost:10001/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "johndoe",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "response_code": "200",
  "response_message": "Success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "name": "John Doe",
      "user_name": "johndoe"
    }
  }
}
```

#### Health Check
```bash
curl -X POST http://localhost:10001/healthz
```

### 🎥 YouTube API Integration

> **Note:** All YouTube API endpoints require authentication. Include the JWT token in the Authorization header.

#### YouTube OAuth2 Authentication

##### Get Authorization URL
```bash
curl -X GET http://localhost:10001/auth/youtube
```

**Response:**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?client_id=...",
  "message": "Visit this URL to authorize the application"
}
```

##### OAuth2 Callback
```bash
# This endpoint is called by Google after user authorization
# The frontend should handle the redirect from Google
curl -X GET "http://localhost:10001/auth/youtube/callback?code=AUTH_CODE&state=STATE"
```

#### Video Operations

##### Get My Videos
```bash
curl -X GET "http://localhost:10001/api/youtube/videos?max_results=10&order=date" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Query Parameters:**
- `max_results` (int): Maximum number of results (1-50, default: 25)
- `page_token` (string): Token for pagination
- `order` (string): Sort order (date, rating, relevance, title, viewCount)
- `published_after` (string): RFC3339 timestamp
- `published_before` (string): RFC3339 timestamp

##### Get Video Details
```bash
curl -X GET http://localhost:10001/api/youtube/videos/VIDEO_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Upload Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "title=My Video Title" \
  -F "description=Video description here" \
  -F "privacy=private" \
  -F "tags=golang,tutorial,api" \
  -F "file=@/path/to/video.mp4"
```

##### Search Videos
```bash
curl -X GET "http://localhost:10001/api/youtube/search?q=golang%20tutorial&max_results=5&type=video" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Query Parameters:**
- `q` (string, required): Search query
- `max_results` (int): Maximum results (1-50, default: 25)
- `page_token` (string): Pagination token
- `order` (string): Sort order (date, rating, relevance, title, viewCount)
- `type` (string): Resource type (video, channel, playlist)
- `channel_id` (string): Search within specific channel
- `duration` (string): Video duration (short, medium, long)
- `definition` (string): Video quality (high, standard)

#### Video Rating Operations

##### Like Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/VIDEO_ID/like \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Dislike Video
```bash
curl -X POST http://localhost:10001/api/youtube/videos/VIDEO_ID/dislike \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Remove Video Rating
```bash
curl -X DELETE http://localhost:10001/api/youtube/videos/VIDEO_ID/rating \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Comment Operations

##### Get Video Comments
```bash
curl -X GET "http://localhost:10001/api/youtube/videos/VIDEO_ID/comments?max_results=20&order=time" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Add Comment
```bash
curl -X POST http://localhost:10001/api/youtube/comments \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "video_id": "VIDEO_ID_HERE",
    "text": "Great video! Thanks for sharing."
  }'
```

##### Update Comment
```bash
curl -X PUT http://localhost:10001/api/youtube/comments/COMMENT_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Updated comment text here"
  }'
```

##### Delete Comment
```bash
curl -X DELETE http://localhost:10001/api/youtube/comments/COMMENT_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Like Comment
```bash
curl -X POST http://localhost:10001/api/youtube/comments/COMMENT_ID/like \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Channel Operations

##### Get My Channel
```bash
curl -X GET http://localhost:10001/api/youtube/channel \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Get Channel Details
```bash
curl -X GET http://localhost:10001/api/youtube/channels/CHANNEL_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Playlist Operations

##### Get My Playlists
```bash
curl -X GET "http://localhost:10001/api/youtube/playlists?max_results=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

##### Create Playlist
```bash
curl -X POST http://localhost:10001/api/youtube/playlists \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My New Playlist",
    "description": "A collection of my favorite videos",
    "privacy": "public"
  }'
```

### 🧪 Test Endpoints (No Authentication Required)

These endpoints are available for testing without authentication:

```bash
# Test: Get My Videos
curl -X GET "http://localhost:10001/test/youtube/videos?max_results=5"

# Test: Search Videos
curl -X GET "http://localhost:10001/test/youtube/search?q=programming&max_results=3"

# Test: Get My Channel
curl -X GET http://localhost:10001/test/youtube/channel
```

## User Management

### Error Responses

All endpoints return standardized error responses:

```json
{
  "response_code": "400",
  "response_message": "Bad Request",
  "error": "Validation failed",
  "details": "user_name is required"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created (for uploads)
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (invalid or missing auth)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource doesn't exist)
- `500` - Internal Server Error

## YouTube API Integration

### Authentication Flow for Frontend

1. **Get Authorization URL:**
   ```typescript
   // Angular service example
   getAuthUrl(): Observable<any> {
     return this.http.get(`${this.baseUrl}/auth/youtube`);
   }
   ```

2. **Redirect User to Google:**
   ```typescript
   authorizeYouTube() {
     this.youtubeService.getAuthUrl().subscribe(response => {
       window.location.href = response.auth_url;
     });
   }
   ```

3. **Handle Callback:**
   ```typescript
   // The callback URL will be handled by your Go backend
   // User will be redirected back to your Angular app with tokens
   ```

### Making Authenticated Requests

```typescript
// Angular HTTP Interceptor example
@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    const token = localStorage.getItem('jwt_token');
    if (token) {
      const authReq = req.clone({
        headers: req.headers.set('Authorization', `Bearer ${token}`)
      });
      return next.handle(authReq);
    }
    return next.handle(req);
  }
}
```

### YouTube API Rate Limits

- **Default quota:** 10,000 units per day
- **Video uploads:** 1600 units each
- **Search operations:** 100 units each
- **Video details:** 1 unit each
- **Comments:** 1-50 units depending on operation

### CORS Configuration

The API supports CORS for frontend integration. Current configuration allows:
- Origins: `https://tulus.tech` (update for your domain)
- Methods: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`
- Headers: `Origin`, `Content-Type`, `Authorization`
- Credentials: Enabled

Update CORS settings in `/server/router.go` for your frontend domain.

## Setup

### Prerequisites
1. Install Go from https://golang.org/dl/
2. Install PostgreSQL from https://www.postgresql.org/download/
3. Install MySQL from https://dev.mysql.com/downloads/mysql/ (optional)
4. Install MongoDB from https://www.mongodb.com/try/download/community (optional)
5. Install Liquibase from https://www.liquibase.org/download
6. Install Docker from https://docs.docker.com/get-docker/
7. Install Docker Compose from https://docs.docker.com/compose/install/

### Environment Setup
1. Copy environment template:
   ```bash
   cp .env.example .env
   cp config.env.example config.env
   ```

2. Configure your environment variables in `.env` and `config.env`

3. For YouTube API integration, follow the detailed setup guide: [YOUTUBE_API_SETUP.md](./YOUTUBE_API_SETUP.md)

### Database Setup
1. Create PostgreSQL database:
   ```bash
   createdb -h localhost -p 5432 -U postgres my_project
   ```

2. Run Liquibase migrations:
  #### PostgreSQL (current changelog used in this repo)
  The project uses an SQL formatted changelog at `liquibase/my_project_postgres/sql/my-project-changelog.sql`.

  Common commands (run from repo root):
  ```bash
  # Preview SQL without applying
  (cd liquibase/my_project_postgres/sql && liquibase updateSQL)

  # Apply all pending changesets
  (cd liquibase/my_project_postgres/sql && liquibase update)

  # Clear and recalculate checksums (only if a committed changeset was edited)
  (cd liquibase/my_project_postgres/sql && liquibase clearChecksums)

  # Roll back last changeset (example)
  (cd liquibase/my_project_postgres/sql && liquibase rollbackCount 1)
  ```

  Properties file (`liquibase/my_project_postgres/sql/liquibase.properties`) supplies:
  ```properties
  changeLogFile=my-project-changelog.sql
  liquibase.command.url=jdbc:postgresql://localhost:5432/my_project
  liquibase.command.username=project
  liquibase.command.password=MyPassword_123
  ```

  #### MySQL (alternative / legacy)
  A MySQL variant changelog exists at `liquibase/my-project/sql/my-project-changelog.sql` (includes share tables).

  ```bash
  # Preview
  (cd liquibase/my-project/sql && liquibase updateSQL)
  # Apply
  (cd liquibase/my-project/sql && liquibase update)
  ```

  NOTE: The application ShareRepository currently uses the MySQL connection; if you decide to switch sharing to PostgreSQL ensure the repository SQL and DSN align.

## Migrations
For full details (adding changesets, rollback, dual-DB strategy) see [docs/MIGRATIONS.md](./docs/MIGRATIONS.md).

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd go-project
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build -o app main.go
   ```

## Usage

### Development Mode
1. Start the server in development mode:
   ```bash
   go run main.go
   ```

2. Or start with hot reload (Air):
  ```bash
  ./start-air.sh
  ```
  This uses the existing `.air.toml` configuration. If Air isn't installed the script installs it (requires Go) or falls back to `go run`.

2. The API will be available at: `http://localhost:10001`

### Production Mode
1. Build and run:
   ```bash
   go build -o app main.go
   ./app
   ```

### Docker Deployment
1. Build and run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

2. Stop the services:
   ```bash
   docker-compose down
   ```

### Database Connection
- **PostgreSQL** (Primary): `localhost:5432`
- **MySQL** (Optional): `localhost:3306` 
- **MongoDB** (Optional): `localhost:27017`

## Angular Frontend Integration

### Service Example
```typescript
// auth.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private baseUrl = 'http://localhost:10001';

  constructor(private http: HttpClient) {}

  register(userData: any): Observable<any> {
    return this.http.post(`${this.baseUrl}/register`, userData);
  }

  login(credentials: any): Observable<any> {
    return this.http.post(`${this.baseUrl}/login`, credentials);
  }

  getAuthHeaders(): HttpHeaders {
    const token = localStorage.getItem('jwt_token');
    return new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });
  }
}

// youtube.service.ts
@Injectable({
  providedIn: 'root'
})
export class YouTubeService {
  private baseUrl = 'http://localhost:10001/api/youtube';

  constructor(private http: HttpClient, private authService: AuthService) {}

  getMyVideos(params?: any): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.get(`${this.baseUrl}/videos`, { headers, params });
  }

  searchVideos(query: string, maxResults = 25): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const params = { q: query, max_results: maxResults };
    return this.http.get(`${this.baseUrl}/search`, { headers, params });
  }

  uploadVideo(videoData: FormData): Observable<any> {
    const token = localStorage.getItem('jwt_token');
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
      // Don't set Content-Type for FormData, let browser set it
    });
    return this.http.post(`${this.baseUrl}/videos/upload`, videoData, { headers });
  }

  likeVideo(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/videos/${videoId}/like`, {}, { headers });
  }

  addComment(videoId: string, text: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const body = { video_id: videoId, text: text };
    return this.http.post(`${this.baseUrl}/comments`, body, { headers });
  }
}
```

### Component Example
```typescript
// video-list.component.ts
export class VideoListComponent implements OnInit {
  videos: any[] = [];
  loading = false;

  constructor(private youtubeService: YouTubeService) {}

  ngOnInit() {
    this.loadMyVideos();
  }

  loadMyVideos() {
    this.loading = true;
    this.youtubeService.getMyVideos({ max_results: 20, order: 'date' })
      .subscribe({
        next: (response) => {
          this.videos = response.data || [];
          this.loading = false;
        },
        error: (error) => {
          console.error('Error loading videos:', error);
          this.loading = false;
        }
      });
  }

  likeVideo(videoId: string) {
    this.youtubeService.likeVideo(videoId)
      .subscribe({
        next: () => {
          console.log('Video liked successfully');
          // Update UI or reload videos
        },
        error: (error) => {
          console.error('Error liking video:', error);
        }
      });
  }
}
```

## Testing

### API Testing with curl
You can test all endpoints using the curl examples provided above. Make sure to:

1. Start the server: `go run main.go`
2. Register a user first
3. Login to get a JWT token
4. Use the JWT token in Authorization headers for protected endpoints

### Example Test Flow
```bash
# 1. Register
curl -X POST http://localhost:10001/register \
  -H "Content-Type: application/json" \
  -d '{"name": "Test User", "user_name": "testuser", "password": "test123"}'

# 2. Login
curl -X POST http://localhost:10001/login \
  -H "Content-Type: application/json" \
  -d '{"user_name": "testuser", "password": "test123"}'

# 3. Use the token from login response in subsequent requests
export JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 4. Test YouTube endpoints
curl -X GET http://localhost:10001/api/youtube/videos \
  -H "Authorization: Bearer $JWT_TOKEN"
```

## License
MIT License

## Manual Local Execution & Debugging

This guide walks through the local MariaDB -> Debezium -> Kafka pipeline from start to finish. It assumes no previous Kafka or Debezium experience and provides commands that can be copied directly into a terminal.

### Architecture and Assumptions

The pipeline has four parts:

1. MariaDB stores the `outbox_events` row written by the Go application.
2. Debezium runs inside Kafka Connect and reads committed MariaDB binlog changes.
3. Kafka receives the routed event on `ledger.events.TransactionAuthorized`.
4. The Go application writes the outbox row with `OUTBOX_STRATEGY=cdc`; it does not run the polling relay in CDC mode.

This repository's application entrypoint is `main.go`, not `cmd/api/main.go`. The API request examples below use `/authorize`; that route must be registered in the HTTP router before the endpoint test can succeed.

The root `docker-compose.yml` provides MariaDB and Kafka. The separate `infrastructure/docker/docker-compose-cdc.yml` provides Kafka Connect and joins the root Compose network. It does not create a second MariaDB or Kafka.

### Prerequisites

Install Docker Desktop with Compose v2, Go, `curl`, Bash, and Make. Verify them from a terminal:

```bash
docker --version

## Real-time Share Status (SSE)

This project provides live share status updates to the frontend using Server-Sent Events (SSE). The Angular app opens a single persistent EventSource connection that streams status changes for all share operations initiated by the authenticated user.
```

Run the commands from the repository root:

```bash
cd /Users/lamboktulussimamora/Projects/issuing-ledger-service
```

The examples use these defaults:

| Component | Local value |
|---|---|
| MariaDB container | `ledger_db` |
| MariaDB database | `ledger` |
| Kafka internal address | `kafka:9093` |
| Kafka Connect REST API | `http://localhost:8083` |
| Go API | `http://localhost:10001` |
| Routed Kafka topic | `ledger.events.TransactionAuthorized` |

### 1. Starting Infrastructure

Open **Terminal 1** and start the existing MariaDB and Kafka services:

```bash

```

Start Kafka Connect in the same terminal:

```bash
### Why SSE?
```

The CDC Compose file uses the external network `issuing-ledger-service_default` by default. If Docker reports that the network does not exist, list networks:

```bash
Shares can take several seconds (external API latency, retries). Instead of polling, the UI updates instantly when:
```

Then retry with the actual root Compose network name:

```bash
1. A share request is accepted (immediate pending or success for track_only mode)
```

Verify the containers:

```bash
2. Background processing finishes (success or failed)
```

You should see MariaDB, Kafka, and a container similar to `issuing-kafka-connect`. Kafka Connect may take a while to become ready even though its container is running.

Confirm that the containers share a network:

```bash
3. A retry attempt updates the attempt_count
```

MariaDB and Kafka must be reachable from Kafka Connect as `mariadb:3306` and `kafka:9093`.

### 2. Checking Kafka Connect Health

Kafka Connect starts a JVM, loads the Debezium plugin, and creates internal Kafka topics. It commonly takes **30-60 seconds** before accepting REST requests.

In **Terminal 1**, poll the REST API:

```bash

```

Ready output begins with HTTP 200 and resembles:

```text
HTTP/1.1 200 OK
Content-Type: application/json

{"version":"3.x.x","commit":"...","kafka_cluster_id":"..."}
```

Before readiness, `curl` may report connection refused. Wait a few seconds and retry. Follow startup logs when debugging:

```bash
### Authentication
```

Press `Ctrl-C` to stop following logs; it does not stop Kafka Connect. The automated equivalent is:

```bash
Browsers cannot set custom Authorization headers on an EventSource, so the backend auth middleware accepts a `auth_token` query parameter that contains the same JWT returned at login.
```

That target retries the REST endpoint for up to two minutes.

### 3. Registering the Connector

Register the connector after the health request returns HTTP 200:

```bash

  -H 'Content-Type: application/json' \
  --data @infrastructure/cdc/outbox-connector.json
```

The first registration returns HTTP 201 and JSON similar to:

```json
{
  "name": "issuing-outbox-connector",
  "config": {
    "connector.class": "io.debezium.connector.mysql.MySqlConnector",
    "database.hostname": "mariadb",
    "table.include.list": "ledger.outbox_events"
  },
  "tasks": [],
  "type": "source"
}
```

The real response includes more configuration fields. HTTP 409 means the connector already exists and is safe to keep using. Check its state:

```bash
```
```

A healthy connector has `"connector": {"state": "RUNNING"}` and at least one task with `"state": "RUNNING"`.

For a failed connector, inspect both the status and logs:

```bash
GET /api/share/stream?auth_token=JWT_TOKEN_HERE
Accept: text/event-stream
```

Typical causes are missing MariaDB binlog settings, missing replication privileges, wrong network membership, or Kafka not being available at `kafka:9093`.

### 4. Booting the Go App

The application must run in CDC mode:

```bash
```
```

Leave this command running in **Terminal 1**. In CDC mode the Go process writes the transactional outbox row and Debezium publishes it. The Go polling relay is deliberately not started.

Open a **new terminal tab, Terminal 2**, and return to the repository:

```bash

```

### 5. Firing the HTTP Request

First check that the API is reachable:

```bash
If the token is invalid or missing the stream responds `401` and closes.
```

Then send an authorization request with a unique idempotency key:

```bash

  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: manual-cdc-test-001' \
  -d '{
    "transaction_id": "manual-cdc-transaction-001",
    "source_account_id": "account-source",
    "destination_account_id": "account-destination",
    "amount": 500,
    "currency": "USD"
  }'
```

An accepted request returns HTTP 200 and resembles:

```json
{
  "transaction_id": "manual-cdc-transaction-001",
  "status": "AUTHORIZED"
}
```

The source and destination accounts must already exist and the source must have at least 500 minor units. Otherwise the application correctly returns a domain error and no successful CDC event is expected.

### 6. Verifying Kafka: The Magic Moment

Open **Terminal 3** so Terminal 1 can keep showing application logs. Start a fresh Kafka consumer before sending a new request:

```bash
### Backend Route & Hub
  --bootstrap-server localhost:9092 \
  --topic ledger.events.TransactionAuthorized \
  --group manual-cdc-verification-$(date +%s) \
  --from-beginning \
  --max-messages 1
```

The consumer waits for one event, prints it, and exits. While it is waiting, use **Terminal 2** to send a request with a new key:

```bash
`main.go` wires `/api/share/stream` through the auth middleware into the share hub (`infrastructure/realtime/share_hub.go`). Each connected user gets an in-memory channel. When share records change, the usecase invokes the broadcaster which fan-outs a JSON event.
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: manual-cdc-test-002' \
  -d '{"transaction_id":"manual-cdc-transaction-002","source_account_id":"account-source","destination_account_id":"account-destination","amount":500,"currency":"USD"}'
```

The consumer should print the raw JSON from the `payload` column, for example:

```json
{"transaction_id":"manual-cdc-transaction-002","amount":500,"currency":"USD"}
```

The exact fields depend on the event JSON written by the Go usecase. Seeing a message on `ledger.events.TransactionAuthorized` after the database transaction commits proves that MariaDB binlog -> Debezium -> Kafka routing worked. No polling worker is involved.

If no message arrives, inspect the connector and database in this order:

```bash

### Event Payload
All events use the SSE event name `share_status` and JSON data:
```json
```

If the outbox row exists but the connector task is `FAILED`, fix the connector, binlog, permissions, or network configuration. If the row does not exist, investigate the API request or account fixtures.

### 7. Clean Teardown

In **Terminal 1**, press `Ctrl-C` to stop the Go application.

Stop Kafka Connect and remove its volumes:

```bash
{
```

Stop MariaDB and Kafka:

```bash
  "type": "share_status",
```

To remove those containers as well:

```bash
  "video_id": "YOUTUBE_VIDEO_ID",
```

The equivalent CDC-only Make target is:

```bash
  "platform": "facebook",
```

`infra-down` removes CDC Compose volumes. Do not use it when you need to preserve Kafka Connect offsets or local data.

### Automated Smoke Test

After the manual flow works, run the automated sequence:

```bash
  "status": "pending | success | failed",
```

It resets CDC infrastructure, starts Kafka Connect, registers the connector, starts the Go process in the background, calls the endpoint, consumes one Kafka message, and cleans up the Go PID with a shell trap. Override defaults when necessary:

```bash
  "external_ref": "<external post id, optional>",
  MARIADB_CONTAINER=ledger_db \
  KAFKA_CONTAINER=kafka \
  CDC_DATABASE_PASSWORD=rootpassword \
  AUTHORIZE_URL=http://localhost:10001/authorize
```
  "error": "<error message if failed>",
  "attempt_count": 1
}
```

Field notes:
- `status` progression (server_post): pending -> (success|failed). For `track_only` mode it is immediately `success` (no background job).
- `attempt_count` increases when a retry is attempted (e.g. transient Facebook errors). The UI can show spinners while pending and badge counts for retries.
- `external_ref` is populated (when available) with the created post ID (e.g. Facebook feed post ID) so the UI can build a link.

### Share Request API
```
POST /api/share
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "video_id": "YOUTUBE_VIDEO_ID",
  "platforms": ["facebook"],
  "mode": "server_post",     // or "track_only"
  "force": false               // set true to re-share even if already success
}
```

Immediate response (example):
```json
[
  {"platform":"facebook","status":"pending","alreadyShared":false}
]
```
An SSE `pending` event is broadcast right away (so UI disables the button & shows spinner). Later the processor sends a final `success` or `failed` event.

### Processing Flow
Text sequence summary:
```
User -> API: POST /api/share (server_post)
API -> DB: Upsert share records (status=pending)
API -> Hub: Broadcast pending event(s)
API -> Worker (goroutine): Process jobs immediately (HTTP post to platform)
Worker -> DB: Update record (success/failed, attempt_count)
Worker -> Hub: Broadcast final status event
UI: Updates platform badge / spinner
```

### Angular Integration (Simplified)
Service snippet illustrating both initiating a share and listening for SSE:
```typescript
import { Injectable, NgZone } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, Subject } from 'rxjs';

export interface ShareStatusEvent {
  type: string;
  video_id: string;
  platform: string;
  status: 'pending' | 'success' | 'failed';
  external_ref?: string;
  error?: string;
  attempt_count?: number;
}

@Injectable({ providedIn: 'root' })
export class SocialShareService {
  private baseUrl = 'http://localhost:10001';
  private evtSource?: EventSource;
  private status$ = new Subject<ShareStatusEvent>();

  constructor(private http: HttpClient, private zone: NgZone) {}

  share(videoId: string, platforms: string[], mode: 'track_only'|'server_post' = 'server_post', force = false): Observable<any> {
    return this.http.post(`${this.baseUrl}/api/share`, { video_id: videoId, platforms, mode, force });
  }

  connectStream() {
    if (this.evtSource) return; // singleton
    const token = localStorage.getItem('jwt_token');
    this.evtSource = new EventSource(`${this.baseUrl}/api/share/stream?auth_token=${token}`);
    this.evtSource.addEventListener('share_status', (e: MessageEvent) => {
      try {
        const data: ShareStatusEvent = JSON.parse(e.data);
        // Ensure change detection via NgZone
        this.zone.run(() => this.status$.next(data));
      } catch {}
    });
    this.evtSource.onerror = () => {
      // Optional: retry with backoff
    };
  }

  statusEvents(): Observable<ShareStatusEvent> { return this.status$.asObservable(); }
}
```

Component usage (outline):
```typescript
ngOnInit() {
  this.shareService.connectStream();
  this.shareService.statusEvents().subscribe(evt => {
    // update UI maps: pendingPlatforms, attempt counts, success/failed badges
  });
}
```

### UI Behavior Recommendations
- Disable a platform button while a `pending` status exists for that (video, platform) tuple.
- Show a spinner + attempt count if `attempt_count > 1`.
- On `failed`, replace spinner with a red badge and enable a "Retry" (force re-share) button.
- On `success`, display external link icon if `external_ref` present.

### Error / Retry Logic
Current implementation performs a single automatic retry for transient Facebook errors (network, 5xx, 429). Future enhancements (exponential backoff, configurable max attempts) can extend this without frontend changes—UI will reflect higher attempt_count values automatically.

### Local Testing Quick Steps
1. Login and copy JWT token.
2. Open browser devtools Network tab.
3. Trigger a share (server_post). Observe immediate SSE `pending` event in the EventStream request.
4. After processing, observe `success` (or `failed`) event and matching UI update.

### Troubleshooting
| Symptom | Likely Cause | Action |
|---------|--------------|--------|
| 401 then closed stream | Missing/invalid auth_token | Ensure JWT is appended: `?auth_token=...` |
| Stuck on pending | External API slow or timeout | Check server logs; verify retries; increase timeouts if needed |
| No events at all | Stream not connected | Call `connectStream()` early (e.g., app init) |

---
