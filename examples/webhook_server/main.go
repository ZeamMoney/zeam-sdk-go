// Example: verify inbound webhooks with HMAC + replay protection.
//
//	go run ./examples/webhook_server
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ZeamMoney/zeam-sdk-go/webhook"
)

func main() {
	secret := os.Getenv("ZEAM_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("set ZEAM_WEBHOOK_SECRET to the webhookSecret.secret captured at registration")
	}

	replay := webhook.NewLRU(10_000)

	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Body is already verified and rewound by the SDK.
		fmt.Fprintln(w, "event accepted")
	})

	h := webhook.Handler(
		app,
		[]byte(secret),
		webhook.WithMaxSkew(5*time.Minute),
		webhook.WithReplayCache(replay),
	)

	mux := http.NewServeMux()
	mux.Handle("/webhooks/zeam", h)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
