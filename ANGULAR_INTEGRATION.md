# Angular Frontend Integration Guide

This guide helps Angular developers integrate with the Go API backend.

## Quick Start

### 1. Install Angular HTTP Client
```bash
npm install @angular/common
```

### 2. Configure HTTP Client in app.module.ts
```typescript
import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';

@NgModule({
  imports: [
    HttpClientModule,
    // ... other imports
  ],
  // ...
})
export class AppModule { }
```

### 3. Create Environment Configuration
```typescript
// src/environments/environment.ts
export const environment = {
  production: false,
  apiUrl: 'http://localhost:10001'
};

// src/environments/environment.prod.ts
export const environment = {
  production: true,
  apiUrl: 'https://your-production-domain.com'
};
```

### 4. Create Authentication Service
```typescript
// src/app/services/auth.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable, BehaviorSubject } from 'rxjs';
import { tap } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface User {
  name: string;
  user_name: string;
}

export interface LoginResponse {
  response_code: string;
  response_message: string;
  data: {
    token: string;
    user: User;
  };
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private baseUrl = environment.apiUrl;
  private currentUserSubject = new BehaviorSubject<User | null>(null);
  public currentUser$ = this.currentUserSubject.asObservable();

  constructor(private http: HttpClient) {
    // Check for existing token on service initialization
    const token = localStorage.getItem('jwt_token');
    const user = localStorage.getItem('current_user');
    if (token && user) {
      this.currentUserSubject.next(JSON.parse(user));
    }
  }

  register(userData: { name: string; user_name: string; password: string }): Observable<any> {
    return this.http.post(`${this.baseUrl}/register`, userData);
  }

  login(credentials: { user_name: string; password: string }): Observable<LoginResponse> {
    return this.http.post<LoginResponse>(`${this.baseUrl}/login`, credentials)
      .pipe(
        tap(response => {
          if (response.response_code === '200' && response.data.token) {
            localStorage.setItem('jwt_token', response.data.token);
            localStorage.setItem('current_user', JSON.stringify(response.data.user));
            this.currentUserSubject.next(response.data.user);
          }
        })
      );
  }

  logout(): void {
    localStorage.removeItem('jwt_token');
    localStorage.removeItem('current_user');
    this.currentUserSubject.next(null);
  }

  getToken(): string | null {
    return localStorage.getItem('jwt_token');
  }

  isAuthenticated(): boolean {
    const token = this.getToken();
    return !!token;
  }

  getAuthHeaders(): HttpHeaders {
    const token = this.getToken();
    return new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });
  }
}
```

### 5. Create YouTube Service
```typescript
// src/app/services/youtube.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { AuthService } from './auth.service';

export interface VideoSearchParams {
  q?: string;
  max_results?: number;
  order?: 'date' | 'rating' | 'relevance' | 'title' | 'viewCount';
  type?: 'video' | 'channel' | 'playlist';
  duration?: 'short' | 'medium' | 'long';
  page_token?: string;
}

export interface VideoListParams {
  max_results?: number;
  order?: 'date' | 'rating' | 'relevance' | 'title' | 'viewCount';
  published_after?: string;
  published_before?: string;
  page_token?: string;
}

@Injectable({
  providedIn: 'root'
})
export class YouTubeService {
  private baseUrl = `${environment.apiUrl}/api/youtube`;

  constructor(
    private http: HttpClient,
    private authService: AuthService
  ) {}

  // OAuth2 Authentication
  getAuthUrl(): Observable<any> {
    return this.http.get(`${environment.apiUrl}/auth/youtube`);
  }

  // Video Operations
  getMyVideos(params?: VideoListParams): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    let httpParams = new HttpParams();
    
    if (params) {
      Object.keys(params).forEach(key => {
        const value = (params as any)[key];
        if (value !== undefined && value !== null) {
          httpParams = httpParams.set(key, value.toString());
        }
      });
    }

    return this.http.get(`${this.baseUrl}/videos`, { headers, params: httpParams });
  }

  getVideoDetails(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.get(`${this.baseUrl}/videos/${videoId}`, { headers });
  }

  searchVideos(params: VideoSearchParams): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    let httpParams = new HttpParams();
    
    Object.keys(params).forEach(key => {
      const value = (params as any)[key];
      if (value !== undefined && value !== null) {
        httpParams = httpParams.set(key, value.toString());
      }
    });

    return this.http.get(`${this.baseUrl}/search`, { headers, params: httpParams });
  }

  uploadVideo(videoData: FormData): Observable<any> {
    const token = this.authService.getToken();
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`
      // Don't set Content-Type for FormData
    });
    return this.http.post(`${this.baseUrl}/videos/upload`, videoData, { headers });
  }

  // Video Rating Operations
  likeVideo(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/videos/${videoId}/like`, {}, { headers });
  }

  dislikeVideo(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/videos/${videoId}/dislike`, {}, { headers });
  }

  removeVideoRating(videoId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.delete(`${this.baseUrl}/videos/${videoId}/rating`, { headers });
  }

  // Comment Operations
  getVideoComments(videoId: string, maxResults = 20): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const params = new HttpParams()
      .set('max_results', maxResults.toString())
      .set('order', 'time');
    
    return this.http.get(`${this.baseUrl}/videos/${videoId}/comments`, { headers, params });
  }

  addComment(videoId: string, text: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const body = { video_id: videoId, text: text };
    return this.http.post(`${this.baseUrl}/comments`, body, { headers });
  }

  updateComment(commentId: string, text: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const body = { text: text };
    return this.http.put(`${this.baseUrl}/comments/${commentId}`, body, { headers });
  }

  deleteComment(commentId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.delete(`${this.baseUrl}/comments/${commentId}`, { headers });
  }

  likeComment(commentId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/comments/${commentId}/like`, {}, { headers });
  }

  // Channel Operations
  getMyChannel(): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.get(`${this.baseUrl}/channel`, { headers });
  }

  getChannelDetails(channelId: string): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.get(`${this.baseUrl}/channels/${channelId}`, { headers });
  }

  // Playlist Operations
  getMyPlaylists(maxResults = 10): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    const params = new HttpParams().set('max_results', maxResults.toString());
    return this.http.get(`${this.baseUrl}/playlists`, { headers, params });
  }

  createPlaylist(playlistData: { title: string; description: string; privacy: string }): Observable<any> {
    const headers = this.authService.getAuthHeaders();
    return this.http.post(`${this.baseUrl}/playlists`, playlistData, { headers });
  }
}
```

### 6. Create HTTP Interceptor for Global Error Handling
```typescript
// src/app/interceptors/auth.interceptor.ts
import { Injectable } from '@angular/core';
import { HttpInterceptor, HttpRequest, HttpHandler, HttpEvent, HttpErrorResponse } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  constructor(
    private authService: AuthService,
    private router: Router
  ) {}

  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    return next.handle(req).pipe(
      catchError((error: HttpErrorResponse) => {
        if (error.status === 401) {
          // Token expired or invalid
          this.authService.logout();
          this.router.navigate(['/login']);
        }
        return throwError(error);
      })
    );
  }
}

// Register in app.module.ts
import { HTTP_INTERCEPTORS } from '@angular/common/http';

@NgModule({
  providers: [
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptor,
      multi: true
    }
  ]
})
export class AppModule { }
```

### 7. Example Component Usage
```typescript
// src/app/components/video-list/video-list.component.ts
import { Component, OnInit } from '@angular/core';
import { YouTubeService, VideoListParams } from '../../services/youtube.service';

@Component({
  selector: 'app-video-list',
  template: `
    <div class="video-list">
      <h2>My Videos</h2>
      
      <div class="controls">
        <input [(ngModel)]="searchQuery" placeholder="Search videos..." />
        <button (click)="searchVideos()" [disabled]="loading">Search</button>
        <button (click)="loadMyVideos()" [disabled]="loading">My Videos</button>
      </div>

      <div *ngIf="loading" class="loading">Loading...</div>
      
      <div class="videos" *ngIf="!loading">
        <div *ngFor="let video of videos" class="video-card">
          <h3>{{ video.title }}</h3>
          <p>{{ video.description }}</p>
          <div class="video-actions">
            <button (click)="likeVideo(video.id)">👍 Like</button>
            <button (click)="dislikeVideo(video.id)">👎 Dislike</button>
            <button (click)="loadComments(video.id)">💬 Comments</button>
          </div>
        </div>
      </div>
    </div>
  `
})
export class VideoListComponent implements OnInit {
  videos: any[] = [];
  loading = false;
  searchQuery = '';

  constructor(private youtubeService: YouTubeService) {}

  ngOnInit() {
    this.loadMyVideos();
  }

  loadMyVideos() {
    this.loading = true;
    const params: VideoListParams = {
      max_results: 20,
      order: 'date'
    };

    this.youtubeService.getMyVideos(params).subscribe({
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

  searchVideos() {
    if (!this.searchQuery.trim()) return;

    this.loading = true;
    this.youtubeService.searchVideos({
      q: this.searchQuery,
      max_results: 10,
      type: 'video'
    }).subscribe({
      next: (response) => {
        this.videos = response.data || [];
        this.loading = false;
      },
      error: (error) => {
        console.error('Error searching videos:', error);
        this.loading = false;
      }
    });
  }

  likeVideo(videoId: string) {
    this.youtubeService.likeVideo(videoId).subscribe({
      next: () => {
        console.log('Video liked successfully');
      },
      error: (error) => {
        console.error('Error liking video:', error);
      }
    });
  }

  dislikeVideo(videoId: string) {
    this.youtubeService.dislikeVideo(videoId).subscribe({
      next: () => {
        console.log('Video disliked successfully');
      },
      error: (error) => {
        console.error('Error disliking video:', error);
      }
    });
  }

  loadComments(videoId: string) {
    this.youtubeService.getVideoComments(videoId).subscribe({
      next: (response) => {
        console.log('Comments:', response.data);
        // Handle comments display
      },
      error: (error) => {
        console.error('Error loading comments:', error);
      }
    });
  }
}
```

### 8. Authentication Guard
```typescript
// src/app/guards/auth.guard.ts
import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

@Injectable({
  providedIn: 'root'
})
export class AuthGuard implements CanActivate {
  constructor(
    private authService: AuthService,
    private router: Router
  ) {}

  canActivate(): boolean {
    if (this.authService.isAuthenticated()) {
      return true;
    } else {
      this.router.navigate(['/login']);
      return false;
    }
  }
}
```

## Error Handling

The API returns standardized error responses:

```typescript
interface ApiError {
  response_code: string;
  response_message: string;
  error?: string;
  details?: string;
}
```

Handle errors consistently:

```typescript
this.youtubeService.getMyVideos().subscribe({
  next: (response) => {
    if (response.response_code === '200') {
      this.videos = response.data;
    } else {
      this.handleError(response);
    }
  },
  error: (httpError) => {
    this.handleHttpError(httpError);
  }
});

private handleError(apiError: ApiError) {
  console.error('API Error:', apiError.response_message);
  // Show user-friendly message
}

private handleHttpError(httpError: any) {
  console.error('HTTP Error:', httpError);
  if (httpError.status === 401) {
    // Redirect to login
  } else if (httpError.status === 500) {
    // Show server error message
  }
}
```

## Testing

Run the Angular development server:
```bash
ng serve
```

The app will be available at `http://localhost:4200` and can communicate with the Go API at `http://localhost:10001`.

## Production Deployment

1. Update environment configuration
2. Build the Angular app: `ng build --prod`
3. Update CORS settings in Go API for production domain
4. Deploy both frontend and backend
