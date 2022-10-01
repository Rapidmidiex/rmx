package user

import "testing"

var jwtKey = []byte("my_secret_key")

var users = map[string]string{
	"user_1": "password_1",
	"user_2": "password_2",
}

func TestAuth(t *testing.T) {

}
