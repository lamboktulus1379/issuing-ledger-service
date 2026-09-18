# YouTube API Setup Guide

This guide will help you set up the YouTube API integration for your Go project.

## Prerequisites

- Google Cloud Console account
- YouTube channel
- Go 1.21 or higher

## Step 1: Google Cloud Console Setup

1. **Go to Google Cloud Console**
   - Visit: https://console.cloud.google.com/
   - Create a new project or select existing one

2. **Enable YouTube Data API v3**
   - Go to APIs & Services → Library
   - Search for "YouTube Data API v3"
   - Click "Enable"

3. **Create OAuth 2.0 Credentials**
   - Go to APIs & Services → Credentials
   - Click "Create Credentials" → "OAuth 2.0 Client IDs"
   - Choose "Web application"
   - Add authorized redirect URIs:
     - `http://localhost:10001/auth/youtube/callback`
   - Add authorized JavaScript origins:
     - `http://localhost:10001`

4. **Download Credentials**
   - Download the JSON file
   - Rename it to something like `client_secret_xxx.json`

## Step 2: Environment Configuration

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Update `.env` with your credentials:**
   ```bash
   # From your downloaded JSON file
   YOUTUBE_CLIENT_ID=your_actual_client_id_from_json
   YOUTUBE_CLIENT_SECRET=your_actual_client_secret_from_json
   YOUTUBE_REDIRECT_URL=http://localhost:10001/auth/youtube/callback
   
   # These will be obtained after OAuth flow
   YOUTUBE_ACCESS_TOKEN=
   YOUTUBE_REFRESH_TOKEN=
   YOUTUBE_CHANNEL_ID=
   
   # Application settings
   SECRET_KEY=your_random_secret_key_for_jwt
   PORT=10001
   ```

## Step 3: First Time Setup

1. **Build and run the application:**
   ```bash
   go build -o youtube-api .
   ./youtube-api
   ```

2. **Get OAuth tokens:**
   - Visit: http://localhost:10001/auth/youtube
   - Follow the OAuth flow
   - Copy the returned tokens

3. **Update environment with tokens:**
   ```bash
   export YOUTUBE_ACCESS_TOKEN="your_access_token"
   export YOUTUBE_REFRESH_TOKEN="your_refresh_token"
   ```

4. **Restart the application:**
   ```bash
   ./youtube-api
   ```

## Step 4: Test the Integration

### Get Your Channel Info
```bash
curl http://localhost:10001/test/youtube/channel
```

### Get Your Videos
```bash
curl "http://localhost:10001/test/youtube/videos?max_results=5"
```

### Search YouTube Videos
```bash
curl "http://localhost:10001/test/youtube/search?q=golang&max_results=3"
```

## Available Endpoints

### Authentication
- `GET /auth/youtube` - Get OAuth authorization URL
- `GET /auth/youtube/callback` - OAuth callback handler

### Video Operations
- `GET /api/youtube/videos` - Get your videos
- `GET /api/youtube/videos/:videoId` - Get video details
- `POST /api/youtube/videos/upload` - Upload video
- `GET /api/youtube/search` - Search videos

### Channel Operations
- `GET /api/youtube/channel` - Get your channel info
- `GET /api/youtube/channels/:channelId` - Get channel details

### Comment Operations
- `GET /api/youtube/videos/:videoId/comments` - Get video comments
- `POST /api/youtube/comments` - Add comment
- `PUT /api/youtube/comments/:commentId` - Update comment
- `DELETE /api/youtube/comments/:commentId` - Delete comment

### Rating Operations
- `POST /api/youtube/videos/:videoId/like` - Like video
- `POST /api/youtube/videos/:videoId/dislike` - Dislike video
- `DELETE /api/youtube/videos/:videoId/rating` - Remove rating

## Security Notes

- Never commit `.env` files to version control
- Keep your client secret secure
- Refresh tokens periodically expire
- Use HTTPS in production
- Implement proper rate limiting

## Troubleshooting

### Common Issues

1. **"OAuth state cookie not found"**
   - Solution: Make sure JavaScript origins match your server port

2. **"Invalid authentication credentials"**
   - Solution: Check if access token has expired, get new tokens

3. **"Redirect URI mismatch"**
   - Solution: Verify redirect URI in Google Cloud Console matches exactly

4. **"API quota exceeded"**
   - Solution: Check your quota limits in Google Cloud Console

### Getting Help

- YouTube Data API Documentation: https://developers.google.com/youtube/v3
- Google Cloud Console: https://console.cloud.google.com/
- OAuth 2.0 Guide: https://developers.google.com/identity/protocols/oauth2
