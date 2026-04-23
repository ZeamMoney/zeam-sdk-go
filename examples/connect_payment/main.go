// Example: run the 9-step Connect off-ramp payment recipe.
//
//	go run ./examples/connect_payment
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ZeamMoney/zeam-sdk-go"
	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/recipes"
	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

func main() {
	ctx := context.Background()
	client, err := zeam.New(zeam.WithEnvironment(zeam.EnvironmentSandbox))
	if err != nil {
		log.Fatal(err)
	}

	// Partner-supplied inputs. In production these come from a secret
	// manager, partner UI, or config file.
	var businessSess *auth.Session // obtained via LoginOTP
	appSeed := os.Getenv("ZEAM_APP_SEED")
	appPub := os.Getenv("ZEAM_APP_PUBLIC_KEY")

	flow := recipes.NewConnectPayment(client, recipes.ConnectPaymentInput{
		BusinessSession:      businessSess,
		ApplicationSeed:      appSeed,
		ApplicationPublicKey: appPub,
		AssociationID:        "6f…e2",
		WalletID:             "wallet-id",
		FundingAsset:         stellar.MustAsset("USDC:GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"),
		BeneficiaryID:        "bb…12",
		Method:               "MOBILE_MONEY",
		CountryISO:           "ZW",
		SendAmount:           "100.00",
	})

	result, err := flow.Do(ctx)
	if err != nil {
		log.Fatalf("connect payment: %v", err)
	}
	fmt.Printf("connect tx=%s status=%s stellar_tx=%s\n",
		result.ConnectTransactionID, result.ConnectStatus, result.StellarTxHash)
}
