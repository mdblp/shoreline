package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// User fields for db store
type User struct {
	ID             string            `json:"userid,omitempty" bson:"userid,omitempty"` // map userid to id
	Username       string            `json:"username,omitempty" bson:"username,omitempty"`
	Emails         []string          `json:"emails,omitempty" bson:"emails,omitempty"`
	Roles          []string          `json:"roles,omitempty" bson:"roles,omitempty"`
	TermsAccepted  string            `json:"termsAccepted,omitempty" bson:"termsAccepted,omitempty"`
	EmailVerified  bool              `json:"emailVerified" bson:"authenticated"` //tag is name `authenticated` for historical reasons
	PwHash         string            `json:"-" bson:"pwhash,omitempty"`
	FailedLogin    *FailedLoginInfos `json:"-" bson:"failedLogin,omitempty"`
	CreatedTime    string            `json:"createdTime,omitempty" bson:"createdTime,omitempty"`
	CreatedUserID  string            `json:"createdUserId,omitempty" bson:"createdUserId,omitempty"`
	ModifiedTime   string            `json:"modifiedTime,omitempty" bson:"modifiedTime,omitempty"`
	ModifiedUserID string            `json:"modifiedUserId,omitempty" bson:"modifiedUserId,omitempty"`
	DeletedTime    string            `json:"deletedTime,omitempty" bson:"deletedTime,omitempty"`
	DeletedUserID  string            `json:"deletedUserId,omitempty" bson:"deletedUserId,omitempty"`
}

// FailedLoginInfos monitor the failed login of an user account.
type FailedLoginInfos struct {
	// Count is the current number of failed login since previous success (reset to 0 after each successful login)
	Count int `json:"-" bson:"count"`
	// Total number of failed login attempt (this value is never reset to 0)
	Total int `json:"-" bson:"total"`
	// Next time we may consider a valid login attempt on this account
	NextLoginAttemptTime string `json:"-" bson:"nextLoginAttemptTime,omitempty"`
}

/*
 * Incoming user details used to create or update a `User`
 */
type NewUserDetails struct {
	Username *string
	Emails   []string
	Password *string
	Roles    []string
}

type NewCustodialUserDetails struct {
	Username *string
	Emails   []string
}

type UpdateUserDetails struct {
	Username      *string
	Emails        []string
	Password      *string
	Roles         []string
	TermsAccepted *string
	EmailVerified *bool
}

const (
	userRoleClinic  = "clinic"
	userRolePatient = "patient"
	userRoleGuest   = "guest"
	userRoleAdmin   = "admin"
)

var (
	errUserDetailsMissing       = errors.New("User details are missing")
	errUserUsernameMissing      = errors.New("Username is missing")
	errUserUsernameInvalid      = errors.New("Username is invalid")
	errUserEmailsMissing        = errors.New("Emails are missing")
	errUserEmailsInvalid        = errors.New("Emails are invalid")
	errUserPasswordMissing      = errors.New("Password is missing")
	errUserPasswordInvalid      = errors.New("Password is invalid")
	errUserRolesInvalid         = errors.New("Roles are invalid")
	errUserTermsAcceptedInvalid = errors.New("Terms accepted is invalid")
	errUserEmailVerifiedInvalid = errors.New("Email verified is invalid")
)

func ExtractBool(data map[string]interface{}, key string) (*bool, bool) {
	if raw, ok := data[key]; !ok {
		return nil, true
	} else if extractedBool, ok := raw.(bool); !ok {
		return nil, false
	} else {
		return &extractedBool, true
	}
}

func ExtractString(data map[string]interface{}, key string) (*string, bool) {
	if raw, ok := data[key]; !ok {
		return nil, true
	} else if extractedString, ok := raw.(string); !ok {
		return nil, false
	} else {
		return &extractedString, true
	}
}

func ExtractArray(data map[string]interface{}, key string) ([]interface{}, bool) {
	if raw, ok := data[key]; !ok {
		return nil, true
	} else if extractedArray, ok := raw.([]interface{}); !ok {
		return nil, false
	} else if len(extractedArray) == 0 {
		return []interface{}{}, true
	} else {
		return extractedArray, true
	}
}

func extractStringArray(data map[string]interface{}, key string) ([]string, bool) {
	var rawArray []interface{}
	var ok bool
	if rawArray, ok = ExtractArray(data, key); !ok {
		return nil, false
	}
	if rawArray == nil {
		return nil, true
	}
	extractedStringArray := make([]string, 0)
	for _, raw := range rawArray {
		var extractedString string
		if extractedString, ok = raw.(string); !ok {
			return nil, false
		}
		extractedStringArray = append(extractedStringArray, extractedString)
	}
	return extractedStringArray, true

}

func ExtractStringMap(data map[string]interface{}, key string) (map[string]interface{}, bool) {
	if raw, ok := data[key]; !ok {
		return nil, true
	} else if extractedMap, ok := raw.(map[string]interface{}); !ok {
		return nil, false
	} else if len(extractedMap) == 0 {
		return map[string]interface{}{}, true
	} else {
		return extractedMap, true
	}
}

func isValidEmail(email string) bool {
	ok, _ := regexp.MatchString(`\A(?i)([^@\s]+)@((?:[-a-z0-9]+\.)+[a-z]{2,})\z`, email)
	return ok
}

func IsValidPassword(password string) bool {
	ok, _ := regexp.MatchString(`\A\S{8,72}\z`, password)
	return ok
}

// isValidRole Verify the user role exists
func isValidRole(role string) bool {
	switch role {
	case userRoleAdmin:
		return true
	case userRoleClinic:
		return true
	case userRoleGuest:
		return true
	case userRolePatient:
		return true
	}
	return false
}

func IsValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func IsValidTimestamp(timestamp string) bool {
	_, err := time.Parse("2006-01-02T15:04:05-07:00", timestamp)
	return err == nil
}

func (details *NewUserDetails) ExtractFromJSON(reader io.Reader) error {
	if reader == nil {
		return errUserDetailsMissing
	}

	var decoded map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		return err
	}

	var (
		username *string
		emails   []string
		password *string
		roles    []string
		ok       bool
	)

	if username, ok = ExtractString(decoded, "username"); !ok {
		return errUserUsernameInvalid
	}
	if emails, ok = extractStringArray(decoded, "emails"); !ok {
		return errUserEmailsInvalid
	}
	if password, ok = ExtractString(decoded, "password"); !ok {
		return errUserPasswordInvalid
	}
	if roles, ok = extractStringArray(decoded, "roles"); !ok {
		return errUserRolesInvalid
	}

	details.Username = username
	details.Emails = emails
	details.Password = password
	details.Roles = roles
	return nil
}

// Validate the new user
func (details *NewUserDetails) Validate() error {
	if details.Username == nil {
		return errUserUsernameMissing
	} else if !isValidEmail(*details.Username) {
		return errUserUsernameInvalid
	}

	if len(details.Emails) == 0 {
		return errUserEmailsMissing
	}
	for _, email := range details.Emails {
		if !isValidEmail(email) {
			return errUserEmailsInvalid
		}
	}

	if details.Password == nil {
		return errUserPasswordMissing
	} else if !IsValidPassword(*details.Password) {
		return errUserPasswordInvalid
	}

	if details.Roles != nil {
		for _, role := range details.Roles {
			fmt.Printf("Testing role: %s\n", role)
			if !isValidRole(role) {
				return errUserRolesInvalid
			}
		}
	}

	return nil
}

func ParseNewUserDetails(reader io.Reader) (*NewUserDetails, error) {
	details := &NewUserDetails{}
	if err := details.ExtractFromJSON(reader); err != nil {
		return nil, err
	}
	return details, nil
}

// NewUser Create the user struct
func NewUser(details *NewUserDetails, salt string) (user *User, err error) {
	if details == nil {
		return nil, errors.New("New user details is nil")
	} else if err := details.Validate(); err != nil {
		return nil, err
	}

	user = &User{Username: *details.Username, Emails: details.Emails, Roles: details.Roles}

	if user.Roles == nil {
		user.Roles = make([]string, 1)
		user.Roles[0] = "patient"
	}

	roles := strings.Join(user.Roles, ";")

	if user.ID, err = generateUniqueHash([]string{*details.Username, *details.Password, roles}, 24); err != nil {
		return nil, errors.New("User: error generating id")
	}

	if err = user.HashPassword(*details.Password, salt); err != nil {
		return nil, errors.New("User: error generating password hash")
	}

	return user, nil
}

func (details *NewCustodialUserDetails) ExtractFromJSON(reader io.Reader) error {
	if reader == nil {
		return errUserDetailsMissing
	}

	var decoded map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		return err
	}

	var (
		username *string
		emails   []string
		ok       bool
	)

	if username, ok = ExtractString(decoded, "username"); !ok {
		return errUserUsernameInvalid
	}
	if emails, ok = extractStringArray(decoded, "emails"); !ok {
		return errUserEmailsInvalid
	}

	details.Username = username
	details.Emails = emails
	return nil
}

func (details *NewCustodialUserDetails) Validate() error {
	if details.Username != nil {
		if !isValidEmail(*details.Username) {
			return errUserUsernameInvalid
		}
	}

	if details.Emails != nil {
		for _, email := range details.Emails {
			if !isValidEmail(email) {
				return errUserEmailsInvalid
			}
		}
	}

	return nil
}

func ParseNewCustodialUserDetails(reader io.Reader) (*NewCustodialUserDetails, error) {
	details := &NewCustodialUserDetails{}
	if err := details.ExtractFromJSON(reader); err != nil {
		return nil, err
	}
	return details, nil
}

func NewCustodialUser(details *NewCustodialUserDetails, salt string) (user *User, err error) {
	if details == nil {
		return nil, errors.New("New custodial user details is nil")
	} else if err := details.Validate(); err != nil {
		return nil, err
	}

	var username string
	if details.Username != nil {
		username = *details.Username
	}

	user = &User{
		Username: username,
		Emails:   details.Emails,
		Roles:    make([]string, 1),
	}
	user.Roles[0] = userRoleGuest // ?

	if user.ID, err = generateUniqueHash([]string{username}, 24); err != nil {
		return nil, errors.New("User: error generating id")
	}

	return user, nil
}

func (details *UpdateUserDetails) ExtractFromJSON(reader io.Reader) error {
	if reader == nil {
		return errUserDetailsMissing
	}

	var decoded map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		return err
	}

	var (
		username      *string
		emails        []string
		password      *string
		roles         []string
		termsAccepted *string
		emailVerified *bool
		ok            bool
	)

	decoded, ok = ExtractStringMap(decoded, "updates")
	if !ok || decoded == nil {
		return errUserDetailsMissing
	}

	if username, ok = ExtractString(decoded, "username"); !ok {
		return errUserUsernameInvalid
	}
	if emails, ok = extractStringArray(decoded, "emails"); !ok {
		return errUserEmailsInvalid
	}
	if password, ok = ExtractString(decoded, "password"); !ok {
		return errUserPasswordInvalid
	}
	if roles, ok = extractStringArray(decoded, "roles"); !ok {
		return errUserRolesInvalid
	}
	if termsAccepted, ok = ExtractString(decoded, "termsAccepted"); !ok {
		return errUserTermsAcceptedInvalid
	}
	if emailVerified, ok = ExtractBool(decoded, "emailVerified"); !ok {
		return errUserEmailVerifiedInvalid
	}

	details.Username = username
	details.Emails = emails
	details.Password = password
	details.Roles = roles
	details.TermsAccepted = termsAccepted
	details.EmailVerified = emailVerified
	return nil
}

func (details *UpdateUserDetails) Validate() error {
	if details.Username != nil {
		if !isValidEmail(*details.Username) {
			return errUserUsernameInvalid
		}
	}

	if details.Emails != nil {
		for _, email := range details.Emails {
			if !isValidEmail(email) {
				return errUserEmailsInvalid
			}
		}
	}

	if details.Password != nil {
		if !IsValidPassword(*details.Password) {
			return errUserPasswordInvalid
		}
	}

	if details.Roles != nil {
		for _, role := range details.Roles {
			if !isValidRole(role) {
				return errUserRolesInvalid
			}
		}
	}

	if details.TermsAccepted != nil {
		if !IsValidTimestamp(*details.TermsAccepted) {
			return errUserTermsAcceptedInvalid
		}
	}

	return nil
}

func ParseUpdateUserDetails(reader io.Reader) (*UpdateUserDetails, error) {
	details := &UpdateUserDetails{}
	if err := details.ExtractFromJSON(reader); err != nil {
		return nil, err
	}
	return details, nil
}

func (u *User) IsDeleted() bool {
	return u.DeletedTime != ""
}

func (u *User) Email() string {
	return u.Username
}

func (u *User) HasRole(role string) bool {
	for _, userRole := range u.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

func (u *User) IsClinic() bool {
	return u.HasRole(userRoleClinic)
}

func (u *User) HashPassword(pw, salt string) error {
	var passwordHash string
	var err error
	if passwordHash, err = GeneratePasswordHash(u.ID, pw, salt); err != nil {
		return err
	}
	u.PwHash = passwordHash
	return nil
}

func (u *User) PasswordsMatch(pw, salt string) bool {
	if u.PwHash == "" || pw == "" {
		return false
	} else if pwMatch, err := GeneratePasswordHash(u.ID, pw, salt); err != nil {
		return false
	} else {
		return u.PwHash == pwMatch
	}
}

func (u *User) IsEmailVerified(secret string) bool {
	if secret != "" {
		if strings.Contains(u.Username, secret) {
			return true
		}
		for i := range u.Emails {
			if strings.Contains(u.Emails[i], secret) {
				return true
			}
		}
	}
	return u.EmailVerified
}

func (u *User) DeepClone() *User {
	clonedUser := &User{
		ID:            u.ID,
		Username:      u.Username,
		TermsAccepted: u.TermsAccepted,
		EmailVerified: u.EmailVerified,
		PwHash:        u.PwHash,
	}
	if u.Emails != nil {
		clonedUser.Emails = make([]string, len(u.Emails))
		copy(clonedUser.Emails, u.Emails)
	}
	if u.Roles != nil {
		clonedUser.Roles = make([]string, len(u.Roles))
		copy(clonedUser.Roles, u.Roles)
	}
	if u.FailedLogin != nil {
		clonedUser.FailedLogin = &FailedLoginInfos{
			Count:                u.FailedLogin.Count,
			Total:                u.FailedLogin.Total,
			NextLoginAttemptTime: u.FailedLogin.NextLoginAttemptTime,
		}
	}
	return clonedUser
}

// CanPerformALogin check if the user can do a login
func (u *User) CanPerformALogin(maxFailedLogin int) bool {
	if u.FailedLogin == nil {
		return true
	}
	if u.FailedLogin.Count < maxFailedLogin {
		return true
	}

	now := time.Now().Format(time.RFC3339)
	if u.FailedLogin.NextLoginAttemptTime < now {
		return true
	}

	return false
}
