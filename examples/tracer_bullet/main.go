// Tracer Bullet — end-to-end integration flow for Zeam API Gateway.
//
// This example walks through the full tracer bullet flow as defined in the
// integration readiness spec:
//
//  1. Login and get a token (SEP-10)
//  2. Get a list of associations
//  3. Register an application with a selected association
//  4. Authenticate and get a token for Connect (SEP-10)
//  5. Query Connect for connectors
//  6. Create or view a beneficiary
//  7. Get connectors for the beneficiary
//  8. Get a quote on the selected connector
//  9. Make payment to Connect using the quote details
//  10. Get the transaction status (retrieve tx hash)
//  11. Execute the quote acceptance with Connect
//
// Required environment variables:
//
//	ZEAM_CLIENT_ID       — Stellar public key (G...)
//	ZEAM_CLIENT_SECRET   — Stellar secret seed (S...)
//
// Usage:
//
//	go run ./examples/tracer_bullet
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ZeamMoney/zeam-sdk-go"
	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/application"
	"github.com/ZeamMoney/zeam-sdk-go/client/connect"
	"github.com/ZeamMoney/zeam-sdk-go/client/payments"
	"github.com/ZeamMoney/zeam-sdk-go/recipes"
)

// tokens holds the authenticated sessions for both tracks.
type tokens struct {
	business *auth.Session // TrackBusiness — OTP/Firebase
	connect  *auth.Session // TrackConnect  — SEP-10
}

func main() {
	ctx := context.Background()

	// ── Construct the SDK client ────────────────────────────────────────
	client, err := zeam.New(
		zeam.WithEnvironment(zeam.EnvironmentProduction),
		zeam.WithVerboseErrors(),
	)
	if err != nil {
		log.Fatalf("sdk init: %v", err)
	}

	var tok tokens

	// ── Step 1a: Login — Business (OTP) ────────────────────────────────
	fmt.Println("\n══ Step 1a: Login — Business (OTP) ══")

	mobile := os.Getenv("ZEAM_MOBILE")
	if mobile == "" {
		log.Fatal("set ZEAM_MOBILE to your E.164 mobile number (e.g. +27821234567)")
	}

	tok.business, err = recipes.LoginOTP(ctx, client, recipes.LoginOTPInput{
		MobileNumber: mobile,
		AskCode: func(ctx context.Context, hint recipes.OTPHint) (string, error) {
			fmt.Printf("  OTP sent to %s — enter code: ", hint.MaskedDestination)
			s := bufio.NewScanner(os.Stdin)
			s.Scan()
			return s.Text(), s.Err()
		},
	})
	if err != nil {
		fmt.Printf("  ✗ Business login failed: %v\n", err)
		fmt.Println("  (continuing with SEP-10 only — check debug output above)")
	} else {
		fmt.Printf("  ✓ Business session — fingerprint=%s track=%s\n", tok.business.Fingerprint(), tok.business.Track())
		fmt.Printf("  token=%s\n", tok.business.IDToken())
	}

	// ── Step 1b: Login — Connect (SEP-10) ──────────────────────────────
	fmt.Println("\n══ Step 1b: Login — Connect (SEP-10) ══")

	stellarSeed := os.Getenv("ZEAM_CLIENT_SECRET")
	stellarPub := os.Getenv("ZEAM_CLIENT_ID")
	if stellarSeed == "" || stellarPub == "" {
		log.Fatal("set ZEAM_CLIENT_ID (G...) and ZEAM_CLIENT_SECRET (S...)")
	}

	tok.connect, err = recipes.ConnectLogin(ctx, client, recipes.ConnectLoginInput{
		StellarSeed: stellarSeed,
		PublicKey:   stellarPub,
	})
	if err != nil {
		log.Fatalf("step 1b connect login: %v", err)
	}
	fmt.Printf("  ✓ Connect session — fingerprint=%s track=%s\n", tok.connect.Fingerprint(), tok.connect.Track())
	fmt.Printf("  token=%s\n", tok.connect.IDToken())

	// Use business session for business endpoints, fall back to connect if OTP failed.
	bizSess := tok.business
	if bizSess == nil {
		fmt.Println("\n  ⚠ No business session — using connect session for business endpoints")
		bizSess = tok.connect
	}

	// ── Step 2: Get a list of associations ──────────────────────────────
	fmt.Println("\n══ Step 2: List associations ══")

	associations, err := client.Business().ListAssociations(ctx, bizSess)
	if err != nil {
		log.Fatalf("step 2 list associations: %v", err)
	}
	if len(associations) == 0 {
		log.Fatal("step 2: no associations found")
	}
	for i, a := range associations {
		fmt.Printf("  [%d] id=%s name=%s\n", i, a.ID, a.AssociationName)
	}
	selectedAssociation := associations[0]
	fmt.Printf("  ✓ Using association: %s (%s)\n", selectedAssociation.AssociationName, selectedAssociation.ID)

	// ── Step 3: Register an application ────────────────────────────────
	fmt.Println("\n══ Step 3: Register application ══")

	appName := fmt.Sprintf("tracer-bullet-g-sdk-%d", time.Now().UnixMilli())

	regResult, err := client.Application().Register(ctx, bizSess, application.RegistrationInput{
		AssociationID:   selectedAssociation.ID,
		ApplicationName: appName,
	})
	if err != nil {
		fmt.Printf("  ✗ Register failed: %v\n", err)
		fmt.Println("  (continuing — see ISSUES.md)")
	} else {
		fmt.Printf("  ⚠ ONE-TIME SECRETS — capture these now!\n")
		fmt.Printf("    stellar.publicKey = %s\n", regResult.Stellar.PublicKey)
		fmt.Printf("    stellar.seed      = %s…\n", regResult.Stellar.Secret[:8])
		fmt.Printf("    connectSecret     = %s…\n", regResult.ConnectSecret[:8])
		fmt.Printf("    apiKey            = %s…\n", regResult.APIKey.Secret[:8])
		fmt.Printf("    webhookSecret     = %s…\n", regResult.WebhookSecret.Secret[:8])
		fmt.Printf("    webhookId         = %s\n", regResult.WebhookSecret.WebhookID)
		fmt.Printf("  ✓ Registered %q — integratorId=%s stellarPub=%s\n",
			appName, regResult.IntegratorID, regResult.Stellar.PublicKey)
	}

	// ── Step 4: Authenticate for Connect (SEP-10) ──────────────────────
	fmt.Println("\n══ Step 4: Connect auth (SEP-10) ══")
	fmt.Printf("  ✓ Connect session already obtained in step 1b — fingerprint=%s\n", tok.connect.Fingerprint())

	// ── Step 5: Query Connect for connectors ────────────────────────────
	fmt.Println("\n══ Step 5: Query connectors ══")

	connectors, err := client.Connect().QueryConnectors(ctx, tok.connect, connect.ConnectorQueryInput{
		CountryISO: "ZA",
		Method:     "CASH",
	})
	if err != nil {
		fmt.Printf("  ✗ Query connectors failed: %v\n", err)
		fmt.Println("  (continuing — see ISSUES.md)")
	} else if len(connectors) == 0 {
		fmt.Println("  ⚠ No connectors found for ZA/CASH")
	} else {
		for i, c := range connectors {
			fmt.Printf("  [%d] id=%s name=%s method=%s active=%t\n", i, c.ID, c.Name, c.Method, c.IsActive)
		}
		fmt.Printf("  ✓ Found %d connector(s)\n", len(connectors))
	}

	// ── Step 6: View beneficiaries ───────────────────────────────────
	fmt.Println("\n══ Step 6: List beneficiaries ══")

	beneficiaries, err := client.Business().ListBeneficiaries(ctx, bizSess, selectedAssociation.ID)
	if err != nil {
		fmt.Printf("  ✗ List beneficiaries failed: %v\n", err)
		fmt.Println("  (continuing — see ISSUES.md)")
	} else if len(beneficiaries) == 0 {
		fmt.Println("  ⚠ No beneficiaries found")
	} else {
		for i, b := range beneficiaries {
			fmt.Printf("  [%d] id=%s destinations=%d\n", i, b.ID, len(b.PaymentDestinations))
		}
		fmt.Printf("  ✓ Found %d beneficiary(ies)\n", len(beneficiaries))

		// ── Step 7: Get connectors for the beneficiary ───────────────────
		fmt.Println("\n══ Step 7: Connectors for beneficiary ══")

		selectedBeneficiary := beneficiaries[0]
		if len(selectedBeneficiary.PaymentDestinations) > 0 {
			dest := selectedBeneficiary.PaymentDestinations[0]
			country := dest.CountryISO
			if country == "" {
				// Destination has no country — fall back to ZA for now.
				country = "ZA"
				fmt.Println("  ⚠ Destination has no country_iso — falling back to ZA")
			}
			fmt.Printf("  Using destination: method=%s country=%s\n", dest.Method, country)

			beneConnectors, err := client.Connect().QueryConnectors(ctx, tok.connect, connect.ConnectorQueryInput{
				CountryISO: country,
				Method:     dest.Method,
			})
			if err != nil {
				fmt.Printf("  ✗ Query connectors for beneficiary failed: %v\n", err)
			} else {
				for i, c := range beneConnectors {
					fmt.Printf("  [%d] id=%s name=%s method=%s active=%t\n", i, c.ID, c.Name, c.Method, c.IsActive)
				}
				fmt.Printf("  ✓ Found %d connector(s) for beneficiary\n", len(beneConnectors))
			}
		} else {
			fmt.Println("  ⚠ Beneficiary has no payment destinations — skipping step 7")
		}
	}

	// ── Step 8: Get a quote on the selected connector ─────────────────
	fmt.Println("\n══ Step 8: Get quote ══")

	// Use the connector from step 5 if available.
	var selectedConnector connect.Connector
	if len(connectors) > 0 {
		selectedConnector = connectors[0]
	}
	if selectedConnector.ID == "" {
		fmt.Println("  ⚠ No connector available — skipping quote")
	} else {
		fmt.Printf("  Using connector: %s (%s)\n", selectedConnector.Name, selectedConnector.ID)

		quote, err := client.Connect().GetQuote(ctx, tok.connect, connect.QuoteInput{
			ConnectorID: selectedConnector.ID,
			Amount:      10,
		})
		if err != nil {
			fmt.Printf("  ✗ Get quote failed: %v\n", err)
			fmt.Println("  (continuing — see ISSUES.md)")
		} else {
			fmt.Printf("  ✓ Quote: id=%s send=%.2f %s receive=%.2f %s fee=%.2f %s total=%.2f expires=%s\n",
				quote.QuoteID, quote.SendAmount, quote.SendCurrency,
				quote.ReceiveAmount, quote.ReceiveCurrency,
				quote.Fee, quote.FeeCurrency, quote.Total, quote.ExpiresAt)

			// ── Step 9: Init transaction + load payment ─────────────────
			fmt.Println("\n══ Step 9: Init + Load transaction ══")

			if quote.FundingInstructions == nil {
				fmt.Println("  ✗ No funding instructions in quote")
			} else {
				funding := quote.FundingInstructions
				fmt.Printf("  Funding: send %.2f %s to %s memo=%s (%s)\n",
					quote.Total, funding.Asset.Code, funding.DestinationAccount,
					funding.Memo[:16]+"…", funding.MemoType)

				// List wallets to find one to pay from.
				wallets, err := client.Business().ListWalletsByAssociation(ctx, bizSess, selectedAssociation.ID)
				if err != nil {
					fmt.Printf("  ✗ List wallets failed: %v\n", err)
				} else if len(wallets) == 0 {
					fmt.Println("  ⚠ No wallets found")
				} else {
					wallet := wallets[0]
					fmt.Printf("  Using wallet: %s (pubkey=%s)\n", wallet.ID, wallet.PublicKey)

					// 9a: Init transaction to get a request ID.
					initResp, err := client.Payments().TransactionInit(ctx, bizSess)
					if err != nil {
						fmt.Printf("  ✗ Transaction init failed: %v\n", err)
					} else {
						fmt.Printf("  ✓ Transaction init — requestId=%s\n", initResp.RequestID)

						// 9b: Load the transaction with payment instructions.
						loadInput := payments.TransactionLoadInput{
							RequestID: initResp.RequestID,
							Instructions: []payments.Instruction{{
								From: payments.FromAccount{
									Account:         wallet.PublicKey,
									AssetCode:       funding.Asset.Code,
									Issuer:          funding.Asset.Issuer,
									AuthorizationID: wallet.VaultID,
								},
								To: payments.ToAccount{
									Account:   funding.DestinationAccount,
									AssetCode: funding.Asset.Code,
									Issuer:    funding.Asset.Issuer,
								},
								Amount: quote.Total,
							}},
							CustomMemo: funding.Memo,
						}
						loadJSON, _ := json.MarshalIndent(loadInput, "  ", "  ")
						fmt.Printf("  [DEBUG] Load payload:\n  %s\n", string(loadJSON))

						loadResp, err := client.Payments().TransactionLoad(ctx, bizSess, loadInput)
						if err != nil {
							fmt.Printf("  ✗ Transaction load failed: %v\n", err)
						} else {
							fmt.Printf("  ✓ Transaction loaded — status=%s requestId=%s\n", loadResp.Status, loadResp.RequestID)

							// ── Step 10: Poll transaction status ───────────────────
							fmt.Println("\n══ Step 10: Transaction status ══")

							var statusResp *payments.TransactionStatusResponse
							for attempt := 1; attempt <= 20; attempt++ {
								statusResp, err = client.Payments().TransactionStatus(ctx, bizSess, initResp.RequestID)
								if err != nil {
									fmt.Printf("  ✗ Get status failed: %v\n", err)
									break
								}
								fmt.Printf("  [%d] status=%s txHash=%s\n", attempt, statusResp.Status, statusResp.TxHash)
								if statusResp.TxHash != "" || statusResp.Status == "Failed" {
									break
								}
								time.Sleep(3 * time.Second)
							}
							if err == nil && statusResp != nil {
								fmt.Printf("  ✓ Final: status=%s txHash=%s\n", statusResp.Status, statusResp.TxHash)

								// ── Step 11: Execute quote acceptance with Connect ─────
								if statusResp.TxHash != "" {
									fmt.Println("\n══ Step 11: Connect execute ══")

									execInput := connect.ExecuteInput{
										Reference:       initResp.RequestID,
										QuoteID:         quote.QuoteID,
										RefundAccount:   connect.RefundAccount{Account: wallet.PublicKey},
										TransactionHash: statusResp.TxHash,
										Beneficiary: &connect.Beneficiary{
											FirstName: "Jacques",
											Lastname:  "Becker",
											Msisdn:    "27783497894",
											IdNumber:  "0207145127081",
										},
									}
									execJSON, _ := json.MarshalIndent(execInput, "  ", "  ")
									fmt.Printf("  [DEBUG] Execute payload:\n  %s\n", string(execJSON))

									execResp, err := client.Connect().Execute(ctx, tok.connect, execInput)
									if err != nil {
										fmt.Printf("  ✗ Connect execute failed: %v\n", err)
									} else {
										fmt.Printf("  ✓ Connect execute — transactionId=%s status=%s\n",
											execResp.TransactionID, execResp.Status)
									}
								} else {
									fmt.Println("  ⚠ No txHash yet — transaction may still be processing")
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Println("\n══ Tracer bullet complete ══")
}
