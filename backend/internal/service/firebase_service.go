package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"customer-help-center-backend/internal/config"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseService holds Firebase Auth client
type FirebaseService struct {
	AuthClient *auth.Client
}

// ServiceAccount represents Firebase service account JSON
type ServiceAccount struct {
	Type                string `json:"type"`
	ProjectID           string `json:"project_id"`
	PrivateKeyID        string `json:"private_key_id"`
	PrivateKey          string `json:"private_key"`
	ClientEmail         string `json:"client_email"`
	ClientID            string `json:"client_id"`
	AuthURI             string `json:"auth_uri"`
	TokenURI            string `json:"token_uri"`
	AuthProviderCertURL string `json:"auth_provider_x509_cert_url"`
	ClientCertURL       string `json:"client_x509_cert_url"`
	UniverseDomain      string `json:"universe_domain"`
}

// NewFirebaseService initializes Firebase Admin SDK
func NewFirebaseService(cfg *config.Config) (*FirebaseService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("firebase config cannot be nil")
	}

	ctx := context.Background()
	var opt option.ClientOption

	if cfg.FirebaseCredentialsJSON != "" {
		credJSON, err := buildServiceAccountJSON(cfg.FirebaseCredentialsJSON)
		if err != nil {
			return nil, err
		}
		opt = option.WithCredentialsJSON(credJSON)

	} else if cfg.FirebaseCredentialsPath != "" {
		opt = option.WithCredentialsFile(cfg.FirebaseCredentialsPath)

	} else {
		return nil, fmt.Errorf("firebase credentials are required")
	}

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase auth client: %w", err)
	}

	log.Println("✅ Firebase Admin SDK initialized successfully")

	return &FirebaseService{
		AuthClient: authClient,
	}, nil
}

// buildServiceAccountJSON validates and normalizes the service account JSON
func buildServiceAccountJSON(raw string) ([]byte, error) {
	var sa ServiceAccount

	if err := json.Unmarshal([]byte(raw), &sa); err != nil {
		return nil, fmt.Errorf("firebase credentials json is invalid")
	}

	// Validate required fields
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("invalid service account type")
	}
	if sa.ProjectID == "" || sa.PrivateKey == "" || sa.ClientEmail == "" || sa.TokenURI == "" {
		return nil, fmt.Errorf("firebase credentials missing required fields")
	}

	// Fix private key newlines
	sa.PrivateKey = strings.ReplaceAll(sa.PrivateKey, `\n`, "\n")

	normalized, err := json.Marshal(sa)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal firebase credentials")
	}

	return normalized, nil
}

// package service

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"strings"

// 	"customer-help-center-backend/internal/config"

// 	firebase "firebase.google.com/go/v4"
// 	"firebase.google.com/go/v4/auth"
// 	"google.golang.org/api/option"
// )

// // FirebaseService holds the Firebase Auth client
// type FirebaseService struct {
// 	AuthClient *auth.Client
// }

// // NewFirebaseService initializes the Firebase Admin SDK
// func NewFirebaseService(cfg *config.Config) (*FirebaseService, error) {
// 	if cfg == nil {
// 		return nil, fmt.Errorf("firebase config cannot be nil")
// 	}

// 	if cfg.FirebaseCredentialsPath == "" && cfg.FirebaseCredentialsJSON == "" {
// 		return nil, fmt.Errorf("firebase credentials are required")
// 	}

// 	ctx := context.Background()

// 	var opt option.ClientOption
// 	if cfg.FirebaseCredentialsJSON != "" {
// 		// Attempt to fix escape sequences in private key if present
// 		var creds map[string]interface{}
// 		if err := json.Unmarshal([]byte(cfg.FirebaseCredentialsJSON), &creds); err == nil {
// 			if pk, ok := creds["private_key"].(string); ok {
// 				creds["private_key"] = strings.ReplaceAll(pk, "\\n", "\n")
// 				if fixedJSON, err := json.Marshal(creds); err == nil {
// 					opt = option.WithCredentialsJSON(fixedJSON)
// 				} else {
// 					opt = option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
// 				}
// 			} else {
// 				opt = option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
// 			}
// 		} else {
// 			opt = option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON))
// 		}
// 	} else {
// 		opt = option.WithCredentialsFile(cfg.FirebaseCredentialsPath)
// 	}

// 	app, err := firebase.NewApp(ctx, nil, opt)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
// 	}

// 	authClient, err := app.Auth(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to initialize firebase auth client: %w", err)
// 	}

// 	log.Println("Firebase Admin SDK initialized successfully")

// 	return &FirebaseService{
// 		AuthClient: authClient,
// 	}, nil
// }
