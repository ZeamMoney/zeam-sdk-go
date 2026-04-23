package zeam_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/ZeamMoney/zeam-sdk-go"
	"github.com/ZeamMoney/zeam-sdk-go/test/fake"
)

func TestNewRejectsPlainHTTPWithoutInsecureOpt(t *testing.T) {
	if _, err := zeam.New(zeam.WithEnvironment(zeam.EnvironmentCustom("http://localhost:8080"))); err == nil {
		t.Fatal("expected error for plain http without WithInsecureTransport")
	}
}

func TestNewRejectsUnsupportedScheme(t *testing.T) {
	if _, err := zeam.New(zeam.WithEnvironment(zeam.EnvironmentCustom("ftp://example.com"))); err == nil {
		t.Fatal("ftp scheme should be rejected")
	}
}

func TestErrorKindFromStatus(t *testing.T) {
	cases := map[int]zeam.Kind{
		400: zeam.KindValidation,
		422: zeam.KindValidation,
		401: zeam.KindAuth,
		403: zeam.KindAuthz,
		404: zeam.KindNotFound,
		409: zeam.KindConflict,
		408: zeam.KindTransient,
		429: zeam.KindTransient,
		503: zeam.KindTransient,
		504: zeam.KindTransient,
		500: zeam.KindRemote,
		502: zeam.KindRemote,
		200: zeam.KindUnknown,
	}
	for code, want := range cases {
		if got := zeam.KindFromStatus(code); got != want {
			t.Errorf("KindFromStatus(%d) = %s want %s", code, got, want)
		}
	}
}

func TestErrorIsMatchesKind(t *testing.T) {
	err := &zeam.Error{Code: "invalid_token", Kind: zeam.KindAuth, Status: 401}
	if !errors.Is(err, zeam.KindAuth) {
		t.Error("errors.Is(err, KindAuth) should be true")
	}
	if errors.Is(err, zeam.KindValidation) {
		t.Error("errors.Is(err, KindValidation) should be false")
	}
}

func TestClientRawUnwrapsEnvelope(t *testing.T) {
	t.Setenv("ZEAM_SDK_ALLOW_INSECURE", "1")

	srv := fake.NewServer([]fake.Route{
		{Method: http.MethodGet, Path: "/v1/echo", Handler: func(w http.ResponseWriter, r *http.Request) {
			fake.WriteEnvelope(w, "rid-1", map[string]string{"hello": "world"})
		}},
	})
	defer srv.Close()

	client, err := zeam.New(
		zeam.WithEnvironment(zeam.EnvironmentCustom(srv.URL())),
		zeam.WithInsecureTransport(),
		zeam.WithSkipVersionCheck(),
	)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	raw, err := client.Raw().GET(context.Background(), "/v1/echo", nil)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["hello"] != "world" {
		t.Errorf("unwrap: got %v want {hello: world}", out)
	}
}
