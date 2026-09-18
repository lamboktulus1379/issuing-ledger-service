# API Documentation - Quick Reference

**Base URL:** `http://localhost:10001`

## 🔑 Authentication Endpoints

### Register User
```bash
POST /register
Content-Type: application/json

{
  "name": "John Doe",
  "user_name": "johndoe", 
  "password": "password123"
}
```

### Login User
```bash
POST /login
Content-Type: application/json

{
  "user_name": "johndoe",
  "password": "password123"
}
```

### Health Check
```bash
POST /healthz
```

## 🎥 YouTube API Endpoints

> **Note:** All YouTube API endpoints require JWT token in Authorization header: `Bearer YOUR_JWT_TOKEN`

### OAuth2 Authentication
```bash
# Get authorization URL
GET /auth/youtube

# OAuth2 callback (handled by Google)
GET /auth/youtube/callback?code=AUTH_CODE&state=STATE
```

### Video Operations
```bash
# Get my videos
GET /api/youtube/videos?max_results=10&order=date

# Get video details
GET /api/youtube/videos/{videoId}

# Upload video
POST /api/youtube/videos/upload
Content-Type: multipart/form-data
- title: "Video Title"
- description: "Video Description"
- privacy: "private" | "public" | "unlisted"
- tags: "tag1,tag2,tag3"
- file: video.mp4

# Search videos
GET /api/youtube/search?q=query&max_results=5&type=video

# Like video
POST /api/youtube/videos/{videoId}/like

# Dislike video  
POST /api/youtube/videos/{videoId}/dislike

# Remove video rating
DELETE /api/youtube/videos/{videoId}/rating
```

### Comment Operations
```bash
# Get video comments
GET /api/youtube/videos/{videoId}/comments?max_results=20&order=time

# Add comment
POST /api/youtube/comments
Content-Type: application/json
{
  "video_id": "VIDEO_ID",
  "text": "Comment text"
}

# Update comment
PUT /api/youtube/comments/{commentId}
Content-Type: application/json
{
  "text": "Updated comment text"
}

# Delete comment
DELETE /api/youtube/comments/{commentId}

# Like comment
POST /api/youtube/comments/{commentId}/like
```

### Channel Operations
```bash
# Get my channel
GET /api/youtube/channel

# Get channel details
GET /api/youtube/channels/{channelId}
```

### Playlist Operations
```bash
# Get my playlists
GET /api/youtube/playlists?max_results=10

# Create playlist
POST /api/youtube/playlists
Content-Type: application/json
{
  "title": "Playlist Title",
  "description": "Playlist Description", 
  "privacy": "public" | "private" | "unlisted"
}
```

## 🧪 Test Endpoints (No Auth Required)
```bash
# Test: Get my videos
GET /test/youtube/videos?max_results=5

# Test: Search videos
GET /test/youtube/search?q=programming&max_results=3

# Test: Get my channel
GET /test/youtube/channel
```

## 📊 Common Query Parameters

### Video List Parameters
- `max_results` (int): 1-50, default: 25
- `page_token` (string): For pagination
- `order` (string): date, rating, relevance, title, viewCount
- `published_after` (string): RFC3339 timestamp
- `published_before` (string): RFC3339 timestamp

### Search Parameters
- `q` (string, required): Search query
- `max_results` (int): 1-50, default: 25
- `page_token` (string): Pagination token
- `order` (string): Sort order
- `type` (string): video, channel, playlist
- `channel_id` (string): Search within channel
- `duration` (string): short, medium, long
- `definition` (string): high, standard

## 📝 Response Format

### Success Response
```json
{
  "response_code": "200",
  "response_message": "Success",
  "data": { ... }
}
```

### Error Response
```json
{
  "response_code": "400",
  "response_message": "Bad Request",
  "error": "Error details"
}
```

## 🔒 Security Headers

All protected endpoints require:
```
Authorization: Bearer YOUR_JWT_TOKEN
Content-Type: application/json
```

## 📊 Rate Limits

YouTube API quotas:
- Default: 10,000 units/day
- Video upload: 1600 units
- Search: 100 units
- Video details: 1 unit
- Comments: 1-50 units

## 🌐 CORS Configuration

Allowed origins: Update in `/server/router.go`
- Methods: GET, POST, PUT, DELETE, PATCH
- Headers: Origin, Content-Type, Authorization
- Credentials: Enabled
