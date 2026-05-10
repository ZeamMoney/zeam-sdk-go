package zeam

// Environment identifies a deployment of the Zeam platform the SDK should
// talk to. Partners normally use one of the pre-declared environments; a
// custom environment (e.g. localhost during development) is declared via
// [EnvironmentCustom].
type Environment struct {
	name    string
	baseURL string
}

// Name returns the human-readable environment name ("production",
// "staging", "sandbox", or "custom").
func (e Environment) Name() string { return e.name }

// BaseURL returns the gateway base URL this environment resolves to.
func (e Environment) BaseURL() string { return e.baseURL }

// Predeclared environments.
var (
	// EnvironmentProduction talks to https://api-gateway.zeam.app.
	//
	// Sandbox mode does not use a separate URL. Access mode is
	// determined by the credentials and account configuration that
	// Zeam applies to your application, not by the URL you call.
	EnvironmentProduction = Environment{name: "production", baseURL: "https://api-gateway.zeam.app"}
)

// EnvironmentCustom declares a custom environment. Typically used for local
// development against a self-hosted gateway (e.g. http://localhost:8080).
//
// Plain-HTTP base URLs are only accepted when [WithInsecureTransport] is set
// AND the ZEAM_SDK_ALLOW_INSECURE environment variable is "1".
func EnvironmentCustom(baseURL string) Environment {
	return Environment{name: "custom", baseURL: baseURL}
}
