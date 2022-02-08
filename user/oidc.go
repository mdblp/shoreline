package user

import (
	"encoding/json"
	"os"
	"time"

	"github.com/caos/oidc/pkg/client/rp"
	httphelper "github.com/caos/oidc/pkg/http"
	"github.com/sirupsen/logrus"
)

// Simple structure to store and exchange OIDC tokens with our frontend
type OidcTokens struct {
	AuthToken    string `json:"auth"`
	RefreshToken string `json:"refresh"`
}

// Encode tokens to JSON (used to send on HTTP responses)
func (t *OidcTokens) Encode() (string, error) {
	if by, err := json.Marshal(t); err != nil {
		return "", err
	} else {
		return string(by), nil
	}
}

//Decode JSON representation of our OIDC tokens into OidcTokens struct
func (t *OidcTokens) Decode(val string) error {
	if err := json.Unmarshal([]byte(val), t); err != nil {
		return err
	} else {
		return nil
	}
}

// Create a OIDC Relying Party object from shoreline configuration
func createOidcProvider(logger *logrus.Logger, cfg *ApiConfig, redirectUrl string) rp.RelyingParty {

	key := []byte(cfg.OAuthAppConfig.Key)
	cookieHandler := httphelper.NewCookieHandler(key, key)
	if os.Getenv("APP_ENV") == "development" {
		cookieHandler = httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure())
	}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithCustomDiscoveryUrl(cfg.OAuthAppConfig.DiscoveryUrl),
	}
	provider, err := rp.NewRelyingPartyOIDC(
		cfg.OAuthAppConfig.IssuerUri,
		cfg.OAuthAppConfig.ClientId,
		cfg.OAuthAppConfig.Secret,
		redirectUrl,
		[]string{"openid", "scope_all"},
		options...)

	if err != nil {
		logger.Fatalf("error creating provider %s", err.Error())
	}
	return provider
}
