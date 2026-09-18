package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

type IFacebookOAuthHandler interface {
	GetAuthURL(ctx *gin.Context)
	Callback(ctx *gin.Context)
	Status(ctx *gin.Context)
	RefreshPages(ctx *gin.Context)
	LinkPage(ctx *gin.Context)
	LinkPageURL(ctx *gin.Context)
}

// oauthState holds per-state metadata so we can reliably associate the OAuth callback
// with the initiating authenticated user. Without persisting userID here, the callback
// would fall back to a placeholder ("demo-user"), causing later page linking attempts
// (which rely on the real user's ID from their JWT) to fail with no_user_token.
type oauthState struct {
	Expiry time.Time
	UserID string
}

type facebookOAuthHandler struct {
	tokenRepo repository.IOAuthToken
	stateMu   sync.Mutex
	states    map[string]oauthState // state -> metadata
}

func NewFacebookOAuthHandler(tokenRepo repository.IOAuthToken) IFacebookOAuthHandler {
	return &facebookOAuthHandler{tokenRepo: tokenRepo, states: map[string]oauthState{}}
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GetAuthURL builds Facebook OAuth URL (user must approve in browser)
func (h *facebookOAuthHandler) GetAuthURL(c *gin.Context) {
	conf := configuration.C.OAuth.Facebook
	if conf.ClientID == "" || conf.RedirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "facebook oauth not configured"})
		return
	}
	state := randomState()
	// Capture the current authenticated user (if any) so callback can store token under correct user_id.
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "demo-user" // development fallback
	}
	h.stateMu.Lock()
	h.states[state] = oauthState{Expiry: time.Now().Add(10 * time.Minute), UserID: userID}
	h.stateMu.Unlock()
	// Comma-separated scopes; we do not urlencode the commas here because Facebook expects raw list.
	scopes := "pages_show_list,pages_read_engagement,pages_manage_posts,public_profile"
	u := url.URL{Scheme: "https", Host: "www.facebook.com", Path: "/v19.0/dialog/oauth"}
	q := u.Query()
	q.Set("client_id", conf.ClientID)
	q.Set("redirect_uri", conf.RedirectURI)
	q.Set("state", state)
	q.Set("scope", scopes)
	u.RawQuery = q.Encode()
	c.JSON(http.StatusOK, gin.H{"auth_url": u.String(), "state": state})
}

// Callback exchanges code for token(s) (placeholder – real FB token exchange not yet implemented)
func (h *facebookOAuthHandler) Callback(c *gin.Context) {
	lg := logger.GetLogger()
	conf := configuration.C.OAuth.Facebook
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	// validate state
	h.stateMu.Lock()
	st, ok := h.states[state]
	if ok && time.Now().After(st.Expiry) { // expired
		ok = false
	}
	if ok {
		delete(h.states, state)
	}
	h.stateMu.Unlock()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}
	// Prefer userID captured at auth-url generation; fallback to current context if present.
	userID := st.UserID
	if ctxUID := c.GetString("user_id"); ctxUID != "" {
		userID = ctxUID
	}

	// 1. Exchange code for short-lived user access token
	tokenURL := fmt.Sprintf("https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s",
		url.QueryEscape(conf.ClientID), url.QueryEscape(conf.RedirectURI), url.QueryEscape(conf.ClientSecret), url.QueryEscape(code))
	shortTokResp, err := http.Get(tokenURL)
	if err != nil {
		lg.Errorf("facebook token exchange request error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "token_request_failed"})
		return
	}
	body, _ := io.ReadAll(shortTokResp.Body)
	shortTokResp.Body.Close()
	if shortTokResp.StatusCode != 200 {
		lg.WithField("body", string(body)).Error("facebook token exchange failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "token_exchange_failed"})
		return
	}
	var shortData struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &shortData); err != nil {
		lg.WithField("err", err).Error("unmarshal short token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_token_failed"})
		return
	}
	// 2. Exchange short-lived for long-lived token
	llURL := fmt.Sprintf("https://graph.facebook.com/v19.0/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
		url.QueryEscape(conf.ClientID), url.QueryEscape(conf.ClientSecret), url.QueryEscape(shortData.AccessToken))
	llResp, err := http.Get(llURL)
	if err != nil {
		lg.Errorf("facebook long-lived exchange error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "long_lived_exchange_failed"})
		return
	}
	llBody, _ := io.ReadAll(llResp.Body)
	llResp.Body.Close()
	if llResp.StatusCode != 200 {
		lg.WithField("body", string(llBody)).Error("long lived token exchange failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "long_lived_token_failed"})
		return
	}
	var llData struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(llBody, &llData); err != nil {
		lg.WithField("err", err).Error("unmarshal long lived token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_long_token_failed"})
		return
	}
	expiresAt := time.Now().Add(time.Duration(llData.ExpiresIn) * time.Second).UTC()

	// 3. Get pages list using long-lived user token
	pagesURL := fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", url.QueryEscape(llData.AccessToken))
	pagesResp, err := http.Get(pagesURL)
	if err != nil {
		lg.Errorf("facebook pages request error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "pages_request_failed"})
		return
	}
	pagesBody, _ := io.ReadAll(pagesResp.Body)
	pagesResp.Body.Close()
	if pagesResp.StatusCode != 200 {
		lg.WithField("body", string(pagesBody)).Error("pages fetch failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "pages_fetch_failed"})
		return
	}
	var pages struct {
		Data []struct {
			Name        string `json:"name"`
			ID          string `json:"id"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(pagesBody, &pages); err != nil {
		lg.WithField("err", err).Error("unmarshal pages list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_pages_failed"})
		return
	}
	if len(pages.Data) == 0 {
		// Store the long-lived USER token so we at least mark connected and can fetch pages later.
		userTokenType := "user"
		tok := &model.OAuthToken{
			UserID:       userID,
			Platform:     "facebook",
			AccessToken:  llData.AccessToken, // user token
			RefreshToken: "",
			ExpiresAt:    &expiresAt,
			Scopes:       "pages_show_list,pages_read_engagement,pages_manage_posts,public_profile",
			TokenType:    &userTokenType,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
			lg.WithFields(map[string]interface{}{
				"error":      err,
				"user_id":    userID,
				"platform":   "facebook",
				"token_type": "user",
			}).Error("failed to upsert facebook user token (no pages). Ensure oauth_tokens table exists and schema matches repository expectations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "store_token_failed"})
			return
		}
		if c.Query("frontend") == "1" {
			c.Header("Content-Type", "text/html; charset=utf-8")
			_, _ = c.Writer.Write([]byte(`<!DOCTYPE html><html><head><title>Facebook Connected (User Only)</title></head><body><script>if (window.opener){window.opener.postMessage({source:'facebook-oauth',connected:true,page_id:null,page_name:null},'*');window.close();}else{document.write('Facebook connected (user token stored, no pages).');}</script></body></html>`))
			return
		}
		c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": nil, "page_name": nil, "info": "user_token_stored_no_pages"})
		return
	}
	// For now auto-select first page; later expose selection UI
	selected := pages.Data[0]
	tokenType := "page"
	scopes := "pages_show_list,pages_read_engagement,pages_manage_posts,public_profile"

	tok := &model.OAuthToken{
		UserID:       userID,
		Platform:     "facebook",
		AccessToken:  selected.AccessToken, // page token used for posting
		RefreshToken: "",                   // facebook page tokens typically long-lived; refresh not used
		ExpiresAt:    &expiresAt,
		Scopes:       scopes,
		PageID:       &selected.ID,
		PageName:     &selected.Name,
		TokenType:    &tokenType,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
		lg.WithFields(map[string]interface{}{
			"error":      err,
			"user_id":    userID,
			"platform":   "facebook",
			"page_id":    selected.ID,
			"page_name":  selected.Name,
			"token_type": "page",
		}).Error("failed to upsert facebook page token. Ensure oauth_tokens table exists and DB user has INSERT/UPDATE privileges")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store_token_failed"})
		return
	}
	if c.Query("frontend") == "1" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		_, _ = c.Writer.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html><head><title>Facebook Connected</title></head><body><script>if (window.opener){window.opener.postMessage({source:'facebook-oauth',connected:true,page_id:'%s',page_name:%q},'*');window.close();}else{document.write('Facebook connected: %s');}</script></body></html>`, selected.ID, selected.Name, selected.Name)))
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": selected.ID, "page_name": selected.Name})
}

// Status returns whether a facebook page token is stored
func (h *facebookOAuthHandler) Status(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"response_code": "401", "response_message": "Unauthorized"})
		return
	}
	tok, err := h.tokenRepo.GetToken(c.Request.Context(), userID, "facebook")
	if err != nil || tok == nil || tok.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}
	resp := gin.H{"connected": true}
	if tok.PageID != nil {
		resp["page_id"] = *tok.PageID
	}
	if tok.PageName != nil {
		resp["page_name"] = *tok.PageName
	}
	c.JSON(http.StatusOK, resp)
}

// RefreshPages attempts to upgrade a stored user token (no pages) to a page token by re-fetching /me/accounts.
// Useful when user had no pages at initial connect but created/was added to a page later.
func (h *facebookOAuthHandler) RefreshPages(c *gin.Context) {
	lg := logger.GetLogger()
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "demo-user"
	}
	tok, err := h.tokenRepo.GetToken(c.Request.Context(), userID, "facebook")
	if err != nil || tok == nil || tok.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_token"})
		return
	}
	// Only proceed if we currently have no page info or token_type == user
	if tok.PageID != nil && *tok.PageID != "" {
		c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": *tok.PageID, "page_name": tok.PageName})
		return
	}
	pagesURL := fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", url.QueryEscape(tok.AccessToken))
	resp, reqErr := http.Get(pagesURL)
	if reqErr != nil {
		lg.WithField("error", reqErr).Error("facebook refresh pages request error")
		c.JSON(http.StatusBadGateway, gin.H{"error": "pages_request_failed"})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		lg.WithField("body", string(body)).Warn("facebook refresh pages fetch failed")
		c.JSON(http.StatusBadGateway, gin.H{"error": "pages_fetch_failed"})
		return
	}
	var pages struct {
		Data []struct {
			Name        string `json:"name"`
			ID          string `json:"id"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &pages); err != nil {
		lg.WithField("err", err).Error("unmarshal refresh pages list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_pages_failed"})
		return
	}
	if len(pages.Data) == 0 {
		c.JSON(http.StatusOK, gin.H{"connected": true, "info": "still_no_pages", "page_id": nil, "page_name": nil})
		return
	}
	selected := pages.Data[0]
	tokenType := "page"
	tok.PageID = &selected.ID
	tok.PageName = &selected.Name
	tok.AccessToken = selected.AccessToken
	tok.TokenType = &tokenType
	now := time.Now().UTC()
	tok.UpdatedAt = now
	if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
		lg.WithFields(map[string]interface{}{"error": err, "user_id": userID, "page_id": selected.ID}).Error("failed to upgrade to page token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store_page_token_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": selected.ID, "page_name": selected.Name, "upgraded": true})
}

// LinkPage allows manually specifying a page_id to fetch its access_token and store it.
// Useful when /me/accounts is empty but you know the page ID (e.g. New Pages Experience delays).
func (h *facebookOAuthHandler) LinkPage(c *gin.Context) {
	var body struct {
		PageID string `json:"page_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_page_id"})
		return
	}
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "demo-user"
	}
	tok, err := h.tokenRepo.GetToken(c.Request.Context(), userID, "facebook")
	if err != nil || tok == nil || tok.AccessToken == "" {
		// Attempt to adopt demo-user token if exists (common when OAuth callback couldn't see user session)
		if userID != "demo-user" {
			if demoTok, derr := h.tokenRepo.GetToken(c.Request.Context(), "demo-user", "facebook"); derr == nil && demoTok != nil && demoTok.AccessToken != "" {
				// clone token under real user id for future operations
				demoTok.UserID = userID
				if upErr := h.tokenRepo.UpsertToken(c.Request.Context(), demoTok); upErr == nil {
					tok = demoTok
				}
			}
		}
		if tok == nil || tok.AccessToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no_user_token"})
			return
		}
	}
	// Fetch page access token
	pageURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=access_token,name&access_token=%s", url.PathEscape(body.PageID), url.QueryEscape(tok.AccessToken))
	resp, reqErr := http.Get(pageURL)
	if reqErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "page_request_failed"})
		return
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "page_fetch_failed", "body": string(b)})
		return
	}
	var pg struct {
		AccessToken string `json:"access_token"`
		Name        string `json:"name"`
	}
	if err := json.Unmarshal(b, &pg); err != nil || pg.AccessToken == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "parse_page_failed"})
		return
	}
	tokenType := "page"
	tok.AccessToken = pg.AccessToken
	tok.PageID = &body.PageID
	tok.PageName = &pg.Name
	tok.TokenType = &tokenType
	now := time.Now().UTC()
	tok.UpdatedAt = now
	if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store_page_token_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": body.PageID, "page_name": pg.Name, "linked": true})
}

// LinkPageURL accepts a page_url, resolves its numeric page id, then behaves like LinkPage.
func (h *facebookOAuthHandler) LinkPageURL(c *gin.Context) {
	var body struct {
		PageURL string `json:"page_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_page_url"})
		return
	}
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "demo-user"
	}
	tok, err := h.tokenRepo.GetToken(c.Request.Context(), userID, "facebook")
	if err != nil || tok == nil || tok.AccessToken == "" {
		if userID != "demo-user" {
			if demoTok, derr := h.tokenRepo.GetToken(c.Request.Context(), "demo-user", "facebook"); derr == nil && demoTok != nil && demoTok.AccessToken != "" {
				demoTok.UserID = userID
				if upErr := h.tokenRepo.UpsertToken(c.Request.Context(), demoTok); upErr == nil {
					tok = demoTok
				}
			}
		}
		if tok == nil || tok.AccessToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no_user_token"})
			return
		}
	}
	// Resolve page id from URL
	resolveURL := fmt.Sprintf("https://graph.facebook.com/v19.0/?id=%s&access_token=%s", url.QueryEscape(body.PageURL), url.QueryEscape(tok.AccessToken))
	rResp, rErr := http.Get(resolveURL)
	if rErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "resolve_request_failed"})
		return
	}
	rBody, _ := io.ReadAll(rResp.Body)
	rResp.Body.Close()
	if rResp.StatusCode != 200 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "resolve_failed", "body": string(rBody)})
		return
	}
	var rData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rBody, &rData); err != nil || rData.ID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "parse_resolve_failed"})
		return
	}
	// Reuse linking logic by constructing synthetic request (duplicate logic for clarity)
	pageURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=access_token,name&access_token=%s", url.PathEscape(rData.ID), url.QueryEscape(tok.AccessToken))
	resp, reqErr := http.Get(pageURL)
	if reqErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "page_request_failed"})
		return
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "page_fetch_failed", "body": string(b)})
		return
	}
	var pg struct {
		AccessToken string `json:"access_token"`
		Name        string `json:"name"`
	}
	if err := json.Unmarshal(b, &pg); err != nil || pg.AccessToken == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "parse_page_failed"})
		return
	}
	tokenType := "page"
	tok.AccessToken = pg.AccessToken
	tok.PageID = &rData.ID
	tok.PageName = &pg.Name
	tok.TokenType = &tokenType
	now := time.Now().UTC()
	tok.UpdatedAt = now
	if err := h.tokenRepo.UpsertToken(c.Request.Context(), tok); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store_page_token_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": true, "page_id": rData.ID, "page_name": pg.Name, "linked": true, "resolved": true})
}
