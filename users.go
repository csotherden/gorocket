package gorocket

import (
	"net/http"
	"github.com/gopackage/ddp"
	"net/url"
	"bytes"
	"errors"
	"crypto/sha256"
	"encoding/hex"
)

type User struct {
	Id       string `json:"_id"`
	UserName string `json:"username"`
}

type UserCredentials struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"pass"`
}

type logoutResponse struct {
	statusResponse
	data struct {
		message string `json:"message"`
	} `json:"data"`
}

type logonResponse struct {
	statusResponse
	Data struct {
		Token  string `json:"authToken"`
		UserId string `json:"userId"`
	} `json:"data"`
}

type ddpLoginRequest struct {
	User     ddpUser `json:"user"`
	Password ddpPassword `json:"password"`
}

type ddpUser struct {
	Email string `json:"email"`
}

type ddpPassword struct {
	Digest    string `json:"digest"`
	Algorithm string `json:"algorithm"`
}

// Login a user. The Email and the Password are mandatory. The auth token of the user is stored in the Rocket instance.
//
// https://rocket.chat/docs/developer-guides/rest-api/authentication/login
func (r *Rocket) Login(credentials UserCredentials) error {
	data := url.Values{"user": {credentials.Email}, "password": {credentials.Password}}
	request, _ := http.NewRequest("POST", r.getUrl()+"/api/v1/login", bytes.NewBufferString(data.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := new(logonResponse)

	if err := r.doRequest(request, response); err != nil {
		return err
	}

	if response.Status == "success" {
		r.auth = &authInfo{id:response.Data.UserId, token:response.Data.Token}
		return nil
	} else {
		return errors.New("Response status: " + response.Status)
	}
}

// Logout a user. The function returns the response message of the server.
//
// https://rocket.chat/docs/developer-guides/rest-api/authentication/logout
func (r *Rocket) Logout() (string, error) {

	if r.auth == nil {
		return "Was not logged in", nil
	}

	request, _ := http.NewRequest("POST", r.getUrl()+"/api/v1/logout", nil)

	response := new(logoutResponse)

	if err := r.doRequest(request, response); err != nil {
		return "", err
	}

	if response.Status == "success" {
		return response.data.message, nil
	} else {
		return "", errors.New("Response status: " + response.Status)
	}
}

// Register a new user on the server. This function does not need a logged in user.
//
// The ddp methods 'registerUser' and 'setUsername' are not documented.
// https://rocket.chat/docs/developer-guides/realtime-api/method-calls/login/
func (r *Rocket) RegisterUser(credentials UserCredentials) error {
	ddpClient, err := r.getDdpClient()
	if err != nil {
		return err;
	}

	if _, err = ddpClient.Call("registerUser", credentials); err != nil {
		return err
	}

	if err = r.ddpLogin(ddpClient, &credentials); err != nil {
		return err
	}

	if _, err = ddpClient.Call("setUsername", credentials.Name); err != nil {
		return err
	}

	return nil
}

func (r *Rocket) ddpLogin(ddpClient *ddp.Client, credentials *UserCredentials) (error) {

	digest := sha256.Sum256([]byte(credentials.Password))

	_, err := ddpClient.Call("login", ddpLoginRequest{
		User:     ddpUser{Email: credentials.Email},
		Password: ddpPassword{Digest: hex.EncodeToString(digest[:]), Algorithm:"sha-256"}})

	return err
}
