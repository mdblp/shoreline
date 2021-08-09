package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type (
	SessionToken struct {
		ID        string `json:"-" bson:"_id"`
		IsServer  bool   `json:"isServer" bson:"isServer"`
		ServerID  string `json:"-" bson:"serverId,omitempty"`
		UserID    string `json:"userId,omitempty" bson:"userId,omitempty"`
		Duration  int64  `json:"-" bson:"duration"`
		ExpiresAt int64  `json:"-" bson:"expiresAt"`
		CreatedAt int64  `json:"-" bson:"createdAt"`
		Time      int64  `json:"-" bson:"time"`
	}

	TokenData struct {
		IsServer     bool   `json:"isserver"`
		UserId       string `json:"userid"`
		Email        string `json:"email,omitempty"`
		Name         string `json:"name,omitempty"`
		Role         string `json:"role,omitempty"`
		DurationSecs int64  `json:"duration"`
		ExpiresAt    int64  `json:"-"`
		Audience     string `json:"audience,omitempty"`
		JwtID        string `json:"-"`
	}

	TokenConfig struct {
		Secret       string
		DurationSecs int64
	}
)

const (
	TOKEN_DURATION_KEY = "tokenduration"
	TP_SESSION_TOKEN   = "x-tidepool-session-token"
	// TP_TRACE_SESSION Session trace: uuid v4
	TP_TRACE_SESSION = "x-tidepool-trace-session"
	signAlg          = "HS256"
)

var (
	errorSessionTokenNoUserID         = errors.New("SessionToken: userId not set")
	errorSessionTokenEmpty            = errors.New("SessionToken: empty token")
	errorSessionTokenInvalid          = errors.New("SessionToken: is invalid")
	errorSessionTokenNoDuration       = errors.New("SessionToken: dur not set")
	errorSessionTokenExpirationNotSet = errors.New("SessionToken: exp not set")
	errorSessionTokenUsrNotSet        = errors.New("SessionToken: usr not set")
	errorSessionTokenJtiNotSet        = errors.New("SessionToken: jti not set")
	errorSessionTokenInvalidSignAlg   = errors.New("SessionToken: invalid sign method")
)

func parseClaims(claims jwt.MapClaims) (*TokenData, error) {
	isServer := claims["svr"] == "yes"
	durationSecs, ok := claims["dur"].(int64)
	if !ok {
		dur64, ok := claims["dur"].(float64)
		if !ok {
			return nil, errorSessionTokenNoDuration
		}
		durationSecs = int64(dur64)
	}

	expiresAt, ok := claims["exp"].(int64)
	if !ok {
		var expiresAtFloat float64
		expiresAtFloat, ok = claims["exp"].(float64)
		if ok {
			expiresAt = int64(expiresAtFloat)
		}
	}
	if !ok || expiresAt <= 0 {
		return nil, errorSessionTokenExpirationNotSet
	}

	userID, ok := claims["usr"].(string)
	if !ok {
		return nil, errorSessionTokenUsrNotSet
	}

	email, ok := claims["email"].(string)
	if !ok {
		email = ""
	}
	name, ok := claims["name"].(string)
	if !ok {
		name = email
	}
	role, ok := claims["role"].(string)
	if !ok {
		role = ""
	}
	jti, ok := claims["jti"].(string)
	if !ok {
		return nil, errorSessionTokenJtiNotSet
	}

	return &TokenData{
		IsServer:     isServer,
		DurationSecs: durationSecs,
		ExpiresAt:    expiresAt,
		UserId:       userID,
		Email:        email,
		Name:         name,
		Role:         role,
		JwtID:        jti,
	}, nil
}

// ParseUnverified just parse the token content, without validating the signature
func ParseUnverified(id string) (*TokenData, error) {
	jwtToken, _, err := new(jwt.Parser).ParseUnverified(id, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	if jwtToken.Method.Alg() != signAlg {
		return nil, errorSessionTokenInvalidSignAlg
	}
	return parseClaims(jwtToken.Claims.(jwt.MapClaims))
}

// UnpackSessionTokenAndVerify verify that the provided token string is valid
func UnpackSessionTokenAndVerify(id string, secret string) (*TokenData, error) {
	if id == "" {
		return nil, errorSessionTokenEmpty
	}

	jwtToken, err := jwt.Parse(id, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
	if err != nil {
		return nil, err
	}
	if !jwtToken.Valid {
		return nil, errorSessionTokenInvalid
	}

	return parseClaims(jwtToken.Claims.(jwt.MapClaims))
}

// CreateSessionToken generate the signed token string from the provided information
func CreateSessionToken(data *TokenData, config *TokenConfig) (*SessionToken, error) {
	if data.UserId == "" {
		return nil, errorSessionTokenNoUserID
	}

	if data.DurationSecs == 0 {
		data.DurationSecs = config.DurationSecs
	}

	now := time.Now()
	createdAt := now.Unix()
	expiresAt := now.Add(time.Duration(data.DurationSecs) * time.Second).Unix()

	jwtToken := jwt.New(jwt.GetSigningMethod(signAlg))
	claims := jwtToken.Claims.(jwt.MapClaims)
	if data.IsServer {
		claims["svr"] = "yes"
	} else {
		claims["svr"] = "no"
	}
	// Add claims specific to our 3rd party services
	if strings.ToUpper(data.Audience) == "ZENDESK" {
		if data.Role == "patient" {
			claims["organization"] = "Patient"
		}
		if data.Role == "hcp" {
			claims["organization"] = "Health professional"
		}
		if data.Role == "caregiver" {
			claims["organization"] = "Patient"
		}
		claims["aud"] = "zendesk"
	} else {
		claims["role"] = data.Role
	}
	claims["usr"] = data.UserId
	if data.Name != "" {
		claims["name"] = data.Name
	}
	if data.Email != "" {
		claims["email"] = data.Email
	}

	claims["dur"] = data.DurationSecs
	claims["exp"] = expiresAt
	claims["iat"] = createdAt
	claims["jti"] = uuid.New()

	tokenString, err := jwtToken.SignedString([]byte(config.Secret))
	if err != nil {
		return nil, err
	}

	sessionToken := &SessionToken{
		ID:        tokenString,
		IsServer:  data.IsServer,
		Duration:  data.DurationSecs,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
		Time:      createdAt,
	}
	if data.IsServer {
		sessionToken.ServerID = data.UserId
	} else {
		sessionToken.UserID = data.UserId
	}

	return sessionToken, nil
}

// ToStringForLog return a string intended to be put in the logs only
func (tokenData *TokenData) ToStringForLog() string {
	var tokenType string
	if tokenData.IsServer {
		tokenType = "srv"
	} else {
		tokenType = "usr"
	}
	return fmt.Sprintf("%s{%s}, jti{%s}, dur{%d}, exp{%v}", tokenType, tokenData.UserId, tokenData.JwtID, tokenData.DurationSecs, tokenData.ExpiresAt)
}
