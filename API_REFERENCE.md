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