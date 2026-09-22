# Go Project - Authentication & Server Management

## 🚀 Quick Start

### Start the Server
```bash
./start-dev.sh    # Development mode with go run
./start-server.sh # Production mode with build
```

### Stop the Server
```bash
./stop-server.sh  # Stops any process on port 10001
```

### Test Authentication
```bash
./test-auth.sh    # Complete authentication flow test
```

## 📋 Project Summary

### ✅ What's Working
- **Go Backend Server**: Running on `http://localhost:10001`
- **User Registration**: Create new users with password hashing
- **JWT Authentication**: Login to get access tokens
- **Protected Routes**: `/api/*` routes require JWT authentication
- **Public Routes**: `/test/*` routes work without authentication
- **CORS Support**: Configured for Angular frontend on ports 4200/4201
- **Database**: PostgreSQL and MongoDB connections working
- **Mock YouTube API**: Returns sample data for testing

### 🔧 Technical Architecture

#### Authentication Flow
1. **Registration**: `POST /register` - Creates user with MD5 hashed password
2. **Login**: `POST /login` - Returns JWT token (5-minute expiry)
3. **Protected Access**: Include `Authorization: Bearer <token>` header
4. **Token Validation**: Middleware verifies JWT signature and user existence

#### Available Endpoints

**Public Endpoints (No Auth Required):**
- `POST /register` - User registration
- `POST /login` - User login
- `POST /healthz` - Health check

## 🧪 Testing Examples

### 1. Create User
```bash
curl -X POST http://localhost:10001/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "user_name": "testuser",
    "password": "testpass123"
  }'
```

### 2. Login (Get JWT Token)
```bash
curl -X POST http://localhost:10001/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "testuser",
    "password": "testpass123"
  }'
```

## 🛠️ Scripts Created

### `start-dev.sh`
- Checks if port 10001 is available
- Kills any existing processes if needed
- Starts server with `go run main.go`
- Includes colored output and error handling

### `start-server.sh`
- Production version that builds the binary first
- Includes logging to `logs/server.log`
- More robust for production deployments

### `stop-server.sh`
- Gracefully stops server processes
- Handles multiple process types (go run, binary)
- Force kills if needed

### `test-auth.sh`
- Complete authentication flow testing
- Creates user, logs in, tests protected/public endpoints
- Returns JWT token for frontend integration

## 🔐 Authentication Details

### JWT Token Structure
```json
{
  "exp": 1754497642,        // Expiration timestamp (5 minutes)
  "is": "6",                // User ID
  "user_name": "authtest"   // Username
}
```

### Middleware Flow
1. Extract `Authorization: Bearer <token>` header
2. Parse JWT token using secret key from config
3. Validate token signature and expiration
4. Lookup user in database
5. Set user context and continue to handler

### Password Security
- Passwords are hashed using MD5 (consider upgrading to bcrypt)
- Hash stored in PostgreSQL database
- Original password never stored in plaintext

### CORS Configuration
The server is configured to accept requests from:
- `http://localhost:4200` (Angular default)
- `http://localhost:4201` (Your current Angular port)
- `https://tulus.tech` (Production domain)

## 🐛 Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   ./stop-server.sh  # Kill existing processes
   ./start-dev.sh    # Start fresh
   ```

2. **Authentication Fails**
   - Check JWT token expiry (5 minutes)
   - Verify Authorization header format: `Bearer <token>`
   - Ensure user exists in database

3. **CORS Errors**
   - Frontend origin must be in allowed list
   - Check server logs for CORS rejection messages

4. **Database Connection**
   - Verify PostgreSQL is running on localhost:5432
   - Check credentials in config.json

### Debug Mode
The server runs in debug mode and shows:
- All registered routes
- Request logs
- Authentication flow details
- Database operation results

## 📁 Project Structure
```
/Users/lamboktulussimamora/Projects/go-project/
├── start-dev.sh          # Development server script
├── start-server.sh       # Production server script  
├── stop-server.sh        # Stop server script
├── test-auth.sh          # Authentication test script
├── config.json           # Main configuration
├── config.env            # Environment variables
├── main.go               # Application entry point
├── server/
│   └── router.go         # Route definitions & CORS
├── interfaces/
│   ├── http/             # HTTP handlers
│   └── middleware/       # Authentication middleware
├── usecase/              # Business logic
├── infrastructure/       # Database, config, utils
└── domain/               # Models and DTOs
```

## 🎯 Next Steps

1. **Frontend Integration**: Use provided JWT token in Angular app
2. **YouTube OAuth**: Set up Google Cloud Console for real YouTube API
3. **Production Deploy**: Use `start-server.sh` for production
4. **Security**: Consider upgrading MD5 to bcrypt for passwords
5. **Monitoring**: Add structured logging and metrics

---

**Status**: ✅ Authentication fully working, server management scripts created, ready for frontend integration!
