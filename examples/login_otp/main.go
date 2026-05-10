// Example: run the Business OTP login end-to-end.
//
//	go run ./examples/login_otp
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ZeamMoney/zeam-sdk-go"
	"github.com/ZeamMoney/zeam-sdk-go/recipes"
)

func main() {
	ctx := context.Background()
	client, err := zeam.New(zeam.WithEnvironment(zeam.EnvironmentProduction))
	if err != nil {
		log.Fatal(err)
	}

	mobile := os.Getenv("ZEAM_DEMO_MOBILE")
	if mobile == "" {
		log.Fatal("set ZEAM_DEMO_MOBILE=+...")
	}

	sess, err := recipes.LoginOTP(ctx, client, recipes.LoginOTPInput{
		MobileNumber: mobile,
		AskCode: func(ctx context.Context, hint recipes.OTPHint) (string, error) {
			fmt.Printf("OTP sent to %s. Enter the code: ", hint.MaskedDestination)
			s := bufio.NewScanner(os.Stdin)
			s.Scan()
			return s.Text(), s.Err()
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("logged in, session fingerprint %s (track=%s)\n", sess.Fingerprint(), sess.Track())
}
