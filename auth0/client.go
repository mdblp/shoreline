package auth0

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mdblp/shoreline/schema"
	"github.com/pkg/errors"
)

type userMetaData struct {
	Role string `json:"role"`
}

type auth0User struct {
	Email         string       `json:"email"`
	EmailVerified bool         `json:"email_verified"`
	UserId        string       `json:"user_id"`
	Metadata      userMetaData `json:"user_metadata"`
}

type auth0Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Auth0Client struct {
	// Client id, secret and url are required to establish a connection to Auth0 and get tokens
	clientId        string
	secret          string
	logger          *log.Logger
	token           string
	mut             sync.Mutex
	closed          chan chan bool // Channel to communicate that the object has been closed
	acquiringToken  bool           // flag set when the serverLoginLoop is running
	refreshInterval time.Duration  // token refresh period in nanoseconds
	userUrl         string
	tokenUrl        string
	audience        string
}

func NewAuth0Client(logger *log.Logger) *Auth0Client {
	auth0Url := os.Getenv("AUTH0_URL")
	auth0Secret := os.Getenv("AUTH0_Secret")
	auth0ClientId := os.Getenv("AUTH0_CLIENT_ID")
	if auth0Url == "" || auth0Secret == "" {
		logger.Fatal("Auth0 configuration not provided")
	}
	interval, _ := time.ParseDuration("1h")
	usrUrl, err := url.Parse(auth0Url + "/api/v2/users")
	if err != nil {
		logger.Fatal("Auth0 configuration incorrect, malformed URL: ", err.Error())
	}
	tokenUrl, _ := url.Parse(auth0Url + "/oauth/token")
	audience, _ := url.Parse(auth0Url + "/api/v2/")

	return &Auth0Client{
		secret:          auth0Secret,
		clientId:        auth0ClientId,
		logger:          logger,
		refreshInterval: interval,
		userUrl:         usrUrl.String(),
		tokenUrl:        tokenUrl.String(),
		audience:        audience.String(),
	}
}

func (client *Auth0Client) Start() error {
	var err error
	if client.secret == "" || client.clientId == "" {
		panic("auth0Client requires a secret and ID to be set")
	}
	if err = client.serverLogin(); err != nil {
		log.Printf("Error on initial server token acquisition, [%v]", err)
		go client.serverLoginLoop(true)
	} else {
		go client.refreshTokenLoop()
	}
	return nil
}

func (client *Auth0Client) serverLoginLoop(launchRefreshTokenLoop bool) {
	var attempts int64
	client.mut.Lock()
	if client.acquiringToken {
		client.mut.Unlock()
		return
	}
	client.acquiringToken = true
	client.mut.Unlock()
	for {
		timer := time.After(client.refreshInterval)
		select {
		case twoWay := <-client.closed:
			twoWay <- true
			return
		case <-timer:
			err := client.serverLogin()
			if err == nil {
				log.Printf("Server token acquired successfully after %v attempts", attempts)
				client.mut.Lock()
				client.acquiringToken = false
				client.mut.Unlock()
				if launchRefreshTokenLoop {
					go client.refreshTokenLoop()
				}
				return
			} else {
				attempts++
				log.Printf("Error when getting server token (attempt %v). Error: %v", attempts, err)
			}
		}
	}
}

func (client *Auth0Client) refreshTokenLoop() {
	for {
		timer := time.After(client.refreshInterval)
		select {
		case twoWay := <-client.closed:
			twoWay <- true
			return
		case <-timer:
			client.mut.Lock()
			acquireInProgress := client.acquiringToken
			client.mut.Unlock()
			if !acquireInProgress {
				if err := client.serverLogin(); err != nil {
					log.Printf("Error on  initial server token refresh, [%v]", err)
					go client.serverLoginLoop(false)
				}
			}
		}
	}
}
func (client *Auth0Client) Close() {
	twoWay := make(chan bool)
	client.closed <- twoWay
	<-twoWay
	client.mut.Lock()
	acquireInProgress := client.acquiringToken
	client.mut.Unlock()
	if acquireInProgress {
		<-twoWay
	}
	client.mut.Lock()
	defer client.mut.Unlock()
	client.token = ""
}

func (client *Auth0Client) serverLogin() error {

	var loginResult auth0Token
	params := strings.Join([]string{
		"grant_type=client_credentials",
		"client_id=" + client.clientId + "",
		"client_secret=" + client.secret,
		"audience=" + client.audience,
	},
		"&")
	payload := strings.NewReader(params)
	req, _ := http.NewRequest("POST", client.tokenUrl, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Error while requesting Auth0: " + err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("Error while requesting Auth0: " + res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&loginResult); err != nil {
		return err
	}

	client.mut.Lock()
	defer client.mut.Unlock()
	client.token = loginResult.AccessToken

	return nil
}

func (client *Auth0Client) GetUser(email string) (*schema.UserData, error) {
	params := url.Values{}
	params.Add("q", "email:\""+email+"\"")
	url, _ := url.Parse(client.userUrl)
	url.RawQuery = params.Encode()
	req, _ := http.NewRequest("GET", url.String(), nil)
	req.Header.Add("authorization", "Bearer "+client.token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failure to get a user")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Error while requesting Auth0: " + res.Status)
	}
	var users []auth0User
	if err := json.NewDecoder(res.Body).Decode(&users); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	user := &schema.UserData{
		UserID:        users[0].UserId,
		Username:      users[0].Email,
		EmailVerified: users[0].EmailVerified,
		Emails:        []string{users[0].Email},
		Roles:         []string{users[0].Metadata.Role},
	}
	return user, nil
}

func (client *Auth0Client) UpdateUser(email string) bool {
	return true
}
