package googlesheet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Printf("Unable to read authorization code: %v\n", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Printf("Unable to retrieve token from web: %v\n", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error closing file")
		}
	}(f)
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Printf("Unable to cache oauth token: %v\n", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("Error closing file: %v\n", err)
		}
	}(f)
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Unable to encode oauth token")
	}
}

func NewGoogleSheet() (*sheets.Service, error) {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Printf("Unable to read client secret file: %v\n", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Printf("Unable to parse client secret file to config: %v\n", err)
		return nil, err
	}
	client := getClient(config)
	log.Printf("%s", "Processing connected to Google PubSub")
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Unable to retrieve Sheets client: %v\n", err)
		return nil, err
	}
	log.Printf("%s", "Google Sheet connected...")
	return srv, nil
}
