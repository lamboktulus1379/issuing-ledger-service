# Configuration Setup Guide

## 🔧 Environment Configuration

This project requires several configuration files for proper operation. **IMPORTANT**: Never commit actual configuration files with real credentials to git.

### 📁 Required Configuration Files

Copy the example files and update them with your actual values:

```bash
# Copy configuration templates
cp config.json.example config.json
cp config.env.example config.env

# For YouTube-specific configuration
cp config.env.example .env.youtube
```

### 🔐 Security Note

The following files are **automatically ignored** by git and should contain your real credentials:

- `config.json` - Main application configuration
- `config.toml` - Alternative TOML configuration
- `config-*.json` - Environment-specific configs
- `*.env` - Environment variables
- `.env.*` - Environment-specific variables
- `client_secret_*.json` - OAuth credentials

### 📝 Configuration Files Guide

#### 1. `config.json`
Main application configuration file. Copy from `config.json.example` and update:

```json
{
    "app": {
        "port": "10001",
        "secretKey": "your-strong-secret-key-here"
    },
    "database": {
        "psql": {
            "name": "your_database",
            "host": "localhost",
            "port": "5432",
            "user": "your_username",
            "password": "your_password"
        }
    },
    "youtube": {
        "apiKey": "your_youtube_api_key",
        "clientId": "your_client_id",
        "clientSecret": "your_client_secret"
    }
}
```

#### 2. `config.env` or `.env.youtube`
Environment variables for sensitive data:

```bash
# Database
DB_NAME=your_database_name
DB_PASSWORD=your_secure_password

# JWT Secret
SECRET_KEY=your-jwt-secret-key

# YouTube API
YOUTUBE_API_KEY=your_api_key
YOUTUBE_CLIENT_ID=your_client_id
YOUTUBE_CLIENT_SECRET=your_client_secret
YOUTUBE_ACCESS_TOKEN=your_access_token
YOUTUBE_REFRESH_TOKEN=your_refresh_token
```

### 🎥 YouTube API Setup

1. **Create Google Cloud Project**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one

2. **Enable YouTube Data API**
   - Navigate to APIs & Services > Library
   - Search for "YouTube Data API v3"
   - Click "Enable"

3. **Create OAuth2 Credentials**
   - Go to APIs & Services > Credentials
   - Click "Create Credentials" > "OAuth 2.0 Client IDs"
   - Choose "Web application"
   - Add authorized redirect URI: `http://localhost:10001/auth/youtube/callback`

4. **Download Credentials**
   - Download the JSON file
   - Rename it to match the pattern in `.gitignore`
   - Extract the `client_id` and `client_secret` for your config

5. **Get API Key**
   - Create an API key in the same credentials section
   - Restrict it to YouTube Data API v3

### 🔄 Getting Access Tokens

For the first time setup:

1. Start the server: `go run main.go`
2. Visit: `http://localhost:10001/auth/youtube`
3. Complete the OAuth flow
4. Copy the access and refresh tokens from the callback
5. Add them to your configuration files

### 🚀 Running the Application

1. **Setup Configuration**
   ```bash
   cp config.json.example config.json
   # Edit config.json with your values
   ```

2. **Set Environment Variables** (optional)
   ```bash
   cp config.env.example .env.youtube
   # Edit .env.youtube with your YouTube credentials
   ```

3. **Start the Server**
   ```bash
   go run main.go
   ```

4. **Test Authentication**
   ```bash
   curl -X POST http://localhost:10001/register \
     -H "Content-Type: application/json" \
     -d '{"name":"Test User","user_name":"testuser","password":"testpass123"}'
   ```

### ⚠️ Important Security Notes

1. **Never commit real credentials** - they're automatically ignored by git
2. **Use strong passwords** and secret keys
3. **Rotate tokens regularly** especially in production
4. **Use environment-specific configs** for different deployment environments
5. **Keep backup** of your configuration files in a secure location

### 🛠️ Troubleshooting

- **Database connection issues**: Check your database credentials and ensure the database is running
- **YouTube API errors**: Verify your API key and OAuth credentials are correct
- **JWT token errors**: Ensure your `secretKey` matches between frontend and backend
- **Port conflicts**: Make sure port 10001 is available or change it in your config

### 📚 Additional Resources

- [YouTube Data API Documentation](https://developers.google.com/youtube/v3)
- [Google Cloud Console](https://console.cloud.google.com/)
- [JWT.io](https://jwt.io/) for debugging JWT tokens
