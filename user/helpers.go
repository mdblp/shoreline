package user

import (
	"encoding/base64"
	"container/list"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const (
	// maxConcurrentLogin Maximum number of concurrent login (TODO: Make this constant as a parameter)
	maxConcurrentLogin int = 100
)

func firstStringNotEmpty(strs ...string) string {
	for _, str := range strs {
		if len(str) > 0 {
			return str
		}
	}
	return ""
}

//Docode the http.Request parsing out the user details
func getGivenDetail(req *http.Request) (d map[string]string) {
	if req.ContentLength > 0 {
		if err := json.NewDecoder(req.Body).Decode(&d); err != nil {
			log.Print(USER_API_PREFIX, "error trying to decode user detail ", err)
			return nil
		}
	}
	return d
}

// Extract the username and password from the authorization
// line of an HTTP header. This function will handle the
// parsing and decoding of the line.
func unpackAuth(authLine string) (usr *User, pw string) {
	if authLine != "" {
		parts := strings.SplitN(authLine, " ", 2)
		payload := parts[1]
		if decodedPayload, err := base64.StdEncoding.DecodeString(payload); err != nil {
			log.Print(USER_API_PREFIX, "Error unpacking authorization header [%s]", err.Error())
		} else {
			details := strings.SplitN(string(decodedPayload), ":", 2)
			if (IsValidEmail(details[0]) || details[0] != "") && IsValidPassword(details[1]) {
				//Note the incoming `name` could infact be id, email or the username
				return &User{Id: details[0], Username: details[0], Emails: []string{details[0]}}, details[1]
			}
		}
	}
	return nil, ""
}

func sendModelAsRes(res http.ResponseWriter, model interface{}) {
	sendModelAsResWithStatus(res, model, http.StatusOK)
}

func sendModelAsResWithStatus(res http.ResponseWriter, model interface{}, statusCode int) {
	res.Header().Set("content-type", "application/json")
	res.WriteHeader(statusCode)

	if jsonDetails, err := json.Marshal(model); err != nil {
		log.Println(USER_API_PREFIX, err.Error())
	} else {
		res.Write(jsonDetails)
	}
}

//send metric
func (a *Api) logMetric(name, token string, params map[string]string) {
	if token == "" {
		a.logger.Println("Missing token so couldn't log metric")
		return
	}
	if params == nil {
		params = make(map[string]string)
	}
	return
}

//send metric
func (a *Api) logMetricAsServer(name, token string, params map[string]string) {
	if token == "" {
		a.logger.Println("Missing token so couldn't log metric")
		return
	}
	if params == nil {
		params = make(map[string]string)
	}
	return
}

//send metric
func (a *Api) logMetricForUser(id, name, token string, params map[string]string) {
	if token == "" {
		a.logger.Println("Missing token so couldn't log metric")
		return
	}
	if params == nil {
		params = make(map[string]string)
	}
	return
}

func (a *Api) sendUser(res http.ResponseWriter, user *User, isServerRequest bool) {
	a.sendUserWithStatus(res, user, http.StatusOK, isServerRequest)
}

func (a *Api) sendUserWithStatus(res http.ResponseWriter, user *User, statusCode int, isServerRequest bool) {
	sendModelAsResWithStatus(res, a.asSerializableUser(user, isServerRequest), statusCode)
}

func (a *Api) sendUsers(res http.ResponseWriter, users []*User, isServerRequest bool) {
	serializables := make([]interface{}, len(users))
	if users != nil {
		for index, user := range users {
			serializables[index] = a.asSerializableUser(user, isServerRequest)
		}
	}
	sendModelAsRes(res, serializables)
}

func (a *Api) asSerializableUser(user *User, isServerRequest bool) interface{} {
	serializable := make(map[string]interface{})
	if len(user.Id) > 0 {
		serializable["userid"] = user.Id
	}
	if len(user.Username) > 0 {
		serializable["username"] = user.Username
	}
	if len(user.Emails) > 0 {
		serializable["emails"] = user.Emails
	}
	if len(user.Roles) > 0 {
		serializable["roles"] = user.Roles
	}
	if len(user.TermsAccepted) > 0 {
		serializable["termsAccepted"] = user.TermsAccepted
	}
	if len(user.Username) > 0 || len(user.Emails) > 0 {
		serializable["emailVerified"] = user.EmailVerified
	}
	if isServerRequest {
		serializable["passwordExists"] = (user.PwHash != "")
	}
	return serializable
}

func (a *Api) appendUserLoginInProgress(user *User) (code int, elem *list.Element) {
	a.loginLimiter.mutex.Lock()
	defer a.loginLimiter.mutex.Unlock()

	// Simple rate limiter
	a.loginLimiter.nInProgress++
	if a.loginLimiter.nInProgress > maxConcurrentLogin {
		return http.StatusTooManyRequests, nil
	}

	for e := a.loginLimiter.userNamesInProgress.Front(); e != nil; e = e.Next() {
		if (e.Value.(User).Username == user.Username) {
			return http.StatusTooManyRequests, nil
		}
	}

	e := a.loginLimiter.userNamesInProgress.PushBack(user)
	return http.StatusOK, e
}

func (a *Api) removeUserLoginInProgress(elem *list.Element) {
	a.loginLimiter.mutex.Lock()

	a.loginLimiter.nInProgress--

	if elem != nil {
		a.loginLimiter.userNamesInProgress.Remove(elem)
	}
	a.loginLimiter.mutex.Unlock()
}
