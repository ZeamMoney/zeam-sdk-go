// Example: register an application and securely capture its one-time
// credentials. The callback shown below panics rather than persisting —
// replace it with a real secret-manager call in production.
//
//	go run ./examples/app_registration
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ZeamMoney/zeam-sdk-go"
	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/application"
	"github.com/ZeamMoney/zeam-sdk-go/recipes"
)

func main() {
	ctx := context.Background()
	client, err := zeam.New(zeam.WithEnvironment(zeam.EnvironmentProduction))
	if err != nil {
		log.Fatal(err)
	}

	// Load an existing Business session. Partners typically drive this
	// via LoginOTP; for a headless test you can rehydrate a session
	// from a cloud secret manager.
	var sess *auth.Session // assume populated by your loader

	result, err := recipes.RegisterApplication(ctx, client, recipes.RegisterAppInput{
		Session: sess,
		Payload: application.RegistrationInput{
			ApplicationName: "demo-app",
			AssociationID:   "your-association-id",
		},
		CaptureOneTimeSecrets: func(ctx context.Context, s recipes.OneTimeSecrets) error {
			// In production: vault.Put(ctx, ...)
			fmt.Printf("CAPTURE NOW — stellar.publicKey=%s, apiKey.keyId=%s, webhook.id=%s\n",
				s.StellarPublicKey, s.APIKey[:4]+"…", s.WebhookID)
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("registered integratorId=%s stellarPublicKey=%s\n", result.IntegratorID, result.StellarPublicKey)
}
