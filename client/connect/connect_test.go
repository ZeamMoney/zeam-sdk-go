package connect

import (
	"context"
	"strings"
	"testing"
)

func TestExecRejectsSSRF(t *testing.T) {
	c := &Client{D: nil, ConnectSecret: "not-used"}
	ctx := context.Background()
	bad := []string{
		"https://evil.com/path",
		"http://evil.com",
		"//evil.com",
		"..",
		"some/../thing",
		"a path",
		"",
	}
	for _, p := range bad {
		if err := c.Exec(ctx, nil, "GET", p, nil, nil); err == nil {
			t.Errorf("Exec(%q) should have failed", p)
		}
	}
}

func TestQueryConnectorsRejectsInvalidEnums(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	_, err := c.QueryConnectors(ctx, nil, ConnectorQueryInput{CountryISO: "zw", Method: "MOBILE_MONEY"})
	if err == nil {
		t.Fatal("lowercase countryISO should have failed validation")
	}
	if !strings.Contains(err.Error(), "countryISO") {
		t.Errorf("unexpected error: %v", err)
	}
}
