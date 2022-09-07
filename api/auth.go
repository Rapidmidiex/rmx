package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"time"
	"unicode"

	"github.com/go-redis/redis/v9"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rog-golang-buddies/rapidmidiex/api/internal/db/user"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	DBCon                *sql.DB
	RedisRefreshTokenDB  *redis.Client
	RedisClientIDDB      *redis.Client
	RedisPasswordTokenDB *redis.Client
}

const (
	authorizationHeader    = "Authorization"
	refreshTokenCookieName = "RMX_DIRECT_RT"
	refreshTokenCookiePath = "/api/v1"
)

type loginRes struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
}

type refreshTokenRes struct {
	AccessToken string `json:"access_token"`
}

type userInfoRes struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (s *AuthService) Login(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request and bind it to UserLoginCreds type
	user := user.User{}
	if err := parse(r, &user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Login the user with provided credentials
	tokens, err := s.login(&user)
	handlerError(w, err)

	rtCookie := http.Cookie{
		Path:     refreshTokenCookiePath,
		Name:     refreshTokenCookieName,
		Value:    tokens.refreshToken,
		HttpOnly: true,
		Secure:   false, // set to true in production
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(refreshTokenExpiry),
	}

	res := loginRes{
		IDToken:     tokens.idToken,
		AccessToken: tokens.accessToken,
	}

	http.SetCookie(w, &rtCookie)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AuthService) Register(w http.ResponseWriter, r *http.Request) {
	user := user.User{}
	if err := parse(r, &user); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := s.register(&user)
	handlerError(w, err)

	w.WriteHeader(http.StatusOK)
}

func (s *AuthService) RefreshToken(w http.ResponseWriter, r *http.Request) {
	rtCookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokens, err := s.refreshToken(rtCookie.Value)
	handlerError(w, err)

	newRTCookie := http.Cookie{
		Path:     refreshTokenCookiePath,
		Name:     refreshTokenCookieName,
		Value:    tokens.refreshToken,
		HttpOnly: true,
		Secure:   false, // set to true in production
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(refreshTokenExpiry),
	}
	res := refreshTokenRes{
		AccessToken: tokens.accessToken,
	}

	http.SetCookie(w, &newRTCookie)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	// remove refresh token cookie
	cookie := &http.Cookie{
		Path:     refreshTokenCookiePath,
		Name:     refreshTokenCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   false, // set to true in production
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	}

	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusOK)
}

// func (s *AuthService) GetUsersEmailList(w http.ResponseWriter, r *http.Request) {
// q := auth.New(s.DBCon)
// users, err := q.ListUsers(context.Background())
// if err != nil {
// log.Println(err)
// w.WriteHeader(http.StatusInternalServerError)
// return
// }

// res := usersEmailListRes{}
// for _, user := range users {
// res.Users = append(res.Users, user.Email)
// }

// w.Header().Set("Content-Type", "application/json")
// w.WriteHeader(http.StatusOK)
// err = json.NewEncoder(w).Encode(res)
// if err != nil {
// log.Println(err)
// w.WriteHeader(http.StatusInternalServerError)
// return
// }
// }

func (s *AuthService) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(emailCtxKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := s.getUserInfo(email)
	handlerError(w, err)

	res := userInfoRes{}
	res.Username = user.Username
	res.Email = user.Email

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AuthService) UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(emailCtxKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newUserInfo := user.User{}
	if err := parse(r, &newUserInfo); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := s.updateUserInfo(email, &newUserInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ------------------------ SERVICE -------------------------------
type (
	authTokens struct {
		idToken      string
		accessToken  string
		refreshToken string
	}
	refreshTokens struct {
		accessToken  string
		refreshToken string
	}
	idTokenClaims struct {
		Email string `json:"email"`
		//	emailVerified bool
	}
	accessTokenClaims struct {
		Email    string `json:"email"`
		ClientID string `json:"client_id"`
	}
	refreshTokenClaims struct {
		Email    string `json:"email"`
		ClientID string `json:"client_id"`
	}
)

const (
	idTokenExpiry      = time.Hour * 10
	accessTokenExpiry  = time.Minute * 5
	refreshTokenExpiry = time.Hour * 24 * 7
)

func (s *AuthService) register(u *user.User) error {
	if isEmptyString(u.Username) {
		return &errInvalidRegisterInfo
	}

	err := validateEmail(u.Email)
	if err != nil {
		return err
	}

	err = validatePassword(u.Password)
	if err != nil {
		return err
	}

	userInfo := user.CreateUserParams{}
	userInfo.Username = u.Username
	userInfo.Email = u.Email
	// u.EmailVerified = false // default value when user registers for the first time
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	userInfo.Password = string(hashedPassword)

	ctx := context.Background()
	q := user.New(s.DBCon)
	_, err = q.CreateUser(ctx, &userInfo)
	var mysqlErr *mysql.MySQLError
	if err != nil {
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return &errUserAlreadyExists
		}
		return err
	}

	return nil
}

func (s *AuthService) login(u *user.User) (authTokens, error) {
	// check if provided email is valid
	err := validateEmail(u.Email)
	if err != nil {
		return authTokens{}, err
	}

	// get user info from database to create new authentication tokens
	ctx := context.Background()
	q := user.New(s.DBCon)
	user, err := q.GetUserByEmail(ctx, u.Email)
	if err != nil {
		return authTokens{}, err
	}

	// check user password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(u.Password))
	if err != nil {
		// if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		// 	err = &models.ErrorResponse{Status: http.StatusUnauthorized, Message: errWrongPassword}
		// }
		// if errors.Is(err, bcrypt.ErrHashTooShort) {
		// 	err = &models.ErrorResponse{Status: http.StatusUnauthorized, Message: errWrongPassword}
		// }
		return authTokens{}, err
	}

	clientID := uuid.New().String()
	idClaims := idTokenClaims{
		Email: user.Email,
		// emailVerified: u.EmailVerified,
	}
	it, err := genIDToken(&idClaims)
	if err != nil {
		return authTokens{}, err
	}

	accessClaims := accessTokenClaims{
		Email:    user.Email,
		ClientID: clientID,
	}
	at, err := genAccessToken(&accessClaims)
	if err != nil {
		return authTokens{}, err
	}

	refreshClaims := refreshTokenClaims{
		Email:    user.Email,
		ClientID: clientID,
	}
	rt, err := genRefreshToken(&refreshClaims)
	if err != nil {
		return authTokens{}, err
	}

	tokens := authTokens{
		idToken:      it,
		accessToken:  at,
		refreshToken: rt,
	}
	return tokens, nil
}

func (s *AuthService) refreshToken(rt string) (refreshTokens, error) {
	// validate and parse the refresh token
	rtPayload, err := parseRefreshTokenWithValidate(rt)
	if err != nil {
		return refreshTokens{}, err
	}

	privateClaims := rtPayload.PrivateClaims()

	email, ok := privateClaims["email"].(string)
	if !ok {
		return refreshTokens{}, &errorResponse{status: http.StatusUnauthorized, message: http.StatusText(http.StatusUnauthorized)}
	}

	// check for token reuse
	reuse, err := s.isTokenUsed(rt)
	if err != nil {
		return refreshTokens{}, err
	}
	if reuse {
		return refreshTokens{}, &errorResponse{status: http.StatusUnauthorized, message: http.StatusText(http.StatusUnauthorized)}
	}

	// save used token in database to detect token reuse when this toke is used again
	// since we have passed the token reuse check, this is the first time the token is being used
	// so we save it in used refresh tokens list to detect future token reuses
	err = s.saveRefreshToken(rt, email, rtPayload.Subject())
	if err != nil {
		return refreshTokens{}, err
	}

	// if the client id is revoked then the token is invalid and is reused by malicious user
	revoked, err := s.isClientIDRevoked(rtPayload.Subject())
	if err != nil {
		return refreshTokens{}, err
	}
	if revoked {
		return refreshTokens{}, &errorResponse{status: http.StatusUnauthorized, message: http.StatusText(http.StatusUnauthorized)}
	}

	// generate new access token from previous access token claims
	newATClaims := accessTokenClaims{
		Email:    email,
		ClientID: rtPayload.Subject(),
	}
	newAT, err := genAccessToken(&newATClaims)
	if err != nil {
		return refreshTokens{}, err
	}

	// generate new refresh token form previous access token claims
	newRTClaims := refreshTokenClaims{
		Email:    email,
		ClientID: rtPayload.Subject(),
	}
	newRT, err := genRefreshToken(&newRTClaims)
	if err != nil {
		return refreshTokens{}, err
	}

	tokens := refreshTokens{
		accessToken:  newAT,
		refreshToken: newRT,
	}

	return tokens, nil
}

func (s *AuthService) saveRefreshToken(token, email, clientID string) error {
	t := refreshTokenClaims{
		Email:    email,
		ClientID: clientID,
	}
	b, err := json.Marshal(&t)
	if err != nil {
		return err
	}
	payload, err := parseRefreshToken(token)
	if err != nil {
		return err
	}
	d := time.Now().UTC().Sub(payload.IssuedAt())

	_, err = s.RedisRefreshTokenDB.Set(context.Background(), token, string(b), d).Result()
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) isTokenUsed(token string) (bool, error) {
	// check if token is available in redis database
	// if it's not then token is not reused
	v, err := s.RedisRefreshTokenDB.Get(context.Background(), token).Result()
	if err != nil {
		switch err {
		case redis.Nil:
			return false, nil
		default:
			return false, err
		}
	}

	// token is available in redis database which means it's reused
	// get token information containing client id and email of user
	t := refreshTokenClaims{}
	err = json.Unmarshal([]byte(v), &t)
	if err != nil {
		return false, err
	}

	// save client id in redis database to deny any refresh token with the sub value of revoked client id
	err = s.revokeClientID(t.ClientID, t.Email)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *AuthService) revokeClientID(clientID, email string) error {
	_, err := s.RedisClientIDDB.Set(context.Background(), clientID, email, refreshTokenExpiry).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) isClientIDRevoked(clientID string) (bool, error) {
	// check if a key with client id exists
	// if the key exists it means that the client id is revoked and token should be denied
	// we don't need the email value here
	_, err := s.RedisClientIDDB.Get(context.Background(), clientID).Result()
	if err != nil {
		switch err {
		case redis.Nil:
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func parseRefreshToken(token string) (payload jwt.Token, err error) {
	payload, err = jwt.Parse([]byte(token),
		jwt.WithKey(jwa.HS256,
			[]byte(viper.GetString("REFRESH_TOKEN_HMAC_SECRET"))))
	return
}

// checks access token validity and returns its payload
func parseAccessTokenWithValidate(token string) (payload jwt.Token, err error) {
	payload, err = jwt.Parse([]byte(token),
		jwt.WithKey(jwa.HS256,
			[]byte(viper.GetString("ACCESS_TOKEN_HMAC_SECRET"))),
		jwt.WithValidate(true))
	return
}

// checks refresh token validity and returns its payload
func parseRefreshTokenWithValidate(token string) (payload jwt.Token, err error) {
	payload, err = jwt.Parse([]byte(token),
		jwt.WithKey(jwa.HS256,
			[]byte(viper.GetString("REFRESH_TOKEN_HMAC_SECRET"))),
		jwt.WithValidate(true))
	return
}

func genIDToken(c *idTokenClaims) (string, error) {
	token, err := jwt.NewBuilder().
		Issuer("http://localhost:8080").
		Subject(c.Email).
		Audience([]string{"http://localhost:3000"}).
		IssuedAt(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(idTokenExpiry)).
		Claim("email", c.Email).
		//		Claim("email_verified", c.emailVerified).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte(viper.GetString("ID_TOKEN_HMAC_SECRET"))))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func genAccessToken(c *accessTokenClaims) (string, error) {
	// scope := strings.Join(c.scope, " ")
	token, err := jwt.NewBuilder().
		Issuer("http://localhost:8080").
		Subject(c.ClientID).
		Audience([]string{"http://localhost:3000"}).
		IssuedAt(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(accessTokenExpiry)).
		Claim("email", c.Email).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte(viper.GetString("ACCESS_TOKEN_HMAC_SECRET"))))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func genRefreshToken(c *refreshTokenClaims) (string, error) {
	token, err := jwt.NewBuilder().
		Issuer("http://localhost:8080").
		Subject(c.ClientID).
		Audience([]string{"http://localhost:3000"}).
		IssuedAt(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(refreshTokenExpiry)).
		Claim("email", c.Email).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte(viper.GetString("REFRESH_TOKEN_HMAC_SECRET"))))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func validateEmail(e string) error {
	if _, err := mail.ParseAddress(e); err != nil {
		return &errInvalidEmail
	}
	return nil
}

func validatePassword(p string) error {
	var (
		hasMinLen = false
		hasNumber = false
	)
	if len(p) >= 8 {
		hasMinLen = true
	}
	for _, char := range p {
		if unicode.IsNumber(char) {
			hasNumber = true
		}
	}

	if !(hasMinLen && hasNumber) {
		return &errBadPassword
	}

	return nil
}

func (s *AuthService) getUserInfo(email string) (user.User, error) {
	q := user.New(s.DBCon)
	userInfo, err := q.GetUserByEmail(context.Background(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

		}
		return user.User{}, err
	}

	return userInfo, nil
}

func (s *AuthService) updateUserInfo(email string, u *user.User) error {
	q := user.New(s.DBCon)
	userInfo, err := q.GetUserByEmail(context.Background(), email)
	if err != nil {
		return err
	}

	updateUser := user.UpdateUserParams{}
	updateUser.ID = userInfo.ID
	if isEmptyString(u.Username) {
		return &errInvalidUpdateInfo
	}
	updateUser.Username = u.Username

	_, err = q.UpdateUser(context.Background(), &updateUser)
	if err != nil {
		return err
	}

	return nil
}
