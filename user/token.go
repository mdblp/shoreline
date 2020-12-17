package user

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type (
	// SessionToken to be stored in database
	SessionToken struct {
		ID        string   `json:"-" bson:"_id"`
		IsServer  bool     `json:"isServer" bson:"isServer"`
		ServerID  string   `json:"-" bson:"serverid,omitempty"`
		UserID    string   `json:"userid,omitempty" bson:"userid,omitempty"`
		Roles     []string `json:"roles,omitempty" bson:"roles,omitempty"`
		ExpiresAt int64    `json:"-" bson:"expiresAt"`
		IssuedAt  int64    `json:"-" bson:"issuedAt"`
		Extended  bool     `json:"-" bson:"extended"`
	}

	// TokenData what is sent in the JWT token
	TokenData struct {
		IsServer     bool     `json:"isserver"`
		UserID       string   `json:"userid"`
		UserRoles    []string `json:"roles"`
		DurationSecs int64    `json:"-"`
	}

	tokenClaims struct {
		IsServer  string   `json:"svr"`
		UserID    string   `json:"usr"`
		UserRoles []string `json:"roles,omitempty"`
		Extended  bool     `json:"ext,omitempty"`
		jwt.StandardClaims
	}

	// TokenConfig from service config
	TokenConfig struct {
		Secret       string
		DurationSecs int64
	}

	mapClaims map[string]interface{}
)

const (
	// TokenDurationKey duration wanted for a login token (HTTP header key)
	TokenDurationKey = "tokenduration"
	tokenSignMethod  = "HS256"
)

var (
	errSessionTokenErrorNoUserID = errors.New("SessionToken: userId not set")
	errSessionTokenInvalid       = errors.New("SessionToken: is invalid")
)

// CreateSessionToken Create a new JWT token
func CreateSessionToken(data *TokenData, config TokenConfig) (*SessionToken, error) {
	if data.UserID == "" {
		return nil, errSessionTokenErrorNoUserID
	}

	extended := false
	if data.DurationSecs == 0 {
		if data.IsServer {
			data.DurationSecs = 24 * 60 * 60
		} else {
			data.DurationSecs = config.DurationSecs
		}
	} else {
		extended = true
	}

	now := time.Now().UTC()
	issuedAt := now.Unix()
	expiresAt := now.Add(time.Duration(data.DurationSecs) * time.Second).Unix()

	claims := &tokenClaims{
		UserID: data.UserID,
	}

	if data.IsServer {
		claims.IsServer = "yes"
	} else {
		claims.IsServer = "no"
		claims.UserRoles = data.UserRoles
	}
	claims.ExpiresAt = expiresAt
	claims.IssuedAt = issuedAt
	if extended {
		claims.Extended = true
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod(tokenSignMethod), claims)

	tokenString, err := token.SignedString([]byte(config.Secret))
	if err != nil {
		return nil, err
	}

	sessionToken := &SessionToken{
		ID:        tokenString,
		IsServer:  data.IsServer,
		ExpiresAt: expiresAt,
		IssuedAt:  issuedAt,
		Extended:  extended,
	}
	if data.IsServer {
		sessionToken.ServerID = data.UserID
	} else {
		sessionToken.UserID = data.UserID
		sessionToken.Roles = data.UserRoles
	}

	return sessionToken, nil
}

func unpackSessionTokenAndVerify(id string, secret string) (*TokenData, error) {
	if id == "" {
		return nil, errSessionTokenErrorNoUserID
	}
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}

	parser := new(jwt.Parser)
	parser.ValidMethods = []string{tokenSignMethod}
	parser.SkipClaimsValidation = false
	parser.UseJSONNumber = true
	jwtToken, err := parser.ParseWithClaims(id, &tokenClaims{}, keyFunc)
	if err != nil {
		return nil, err
	}
	if !jwtToken.Valid {
		return nil, errSessionTokenInvalid
	}

	claims := jwtToken.Claims.(*tokenClaims)

	if !claims.VerifyExpiresAt(time.Now().UTC().Unix(), true) {
		return nil, errSessionTokenInvalid
	}

	isServer := claims.IsServer == "yes"
	userID := claims.UserID
	issuedAt := claims.IssuedAt
	expiresAt := claims.ExpiresAt
	durationSecs := expiresAt - issuedAt

	tokenData := &TokenData{
		IsServer:     isServer,
		UserID:       userID,
		DurationSecs: durationSecs,
	}

	if !isServer {
		tokenData.UserRoles = claims.UserRoles
	}

	return tokenData, nil
}

func extractTokenDuration(r *http.Request) int64 {

	durString := r.Header.Get(TokenDurationKey)

	if durString != "" {
		//if there is an error we just return a duration of zero
		dur, err := strconv.ParseInt(durString, 10, 64)
		if err == nil {
			return dur
		}
	}
	return 0
}

func hasServerToken(tokenString, secret string) bool {
	td, err := unpackSessionTokenAndVerify(tokenString, secret)
	if err != nil {
		return false
	}
	return td.IsServer
}
