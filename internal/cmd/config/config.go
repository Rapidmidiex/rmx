package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type StoreConfig struct {
	DatabaseURL string
	NatsURL     string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
}

type ProvidersConfig struct {
	Google GoogleConfig
}

type Keys struct {
	CookieHashKey       string
	CookieEncryptionKey string
	JWTPrivateKey       *ecdsa.PrivateKey
	JWTPublicKey        *ecdsa.PublicKey
}

type AuthConfig struct {
	Enable    bool
	Providers ProvidersConfig
	Keys      Keys
}

type Config struct {
	Server ServerConfig
	Store  StoreConfig
	Auth   AuthConfig
	Dev    bool
}

func LoadFromEnv() *Config {
	if err := godotenv.Load("rmx.env", ".env"); err != nil {
		log.Fatalf("rmx: couldn't read env\n%v", err)
	}

	// server
	serverPort := readEnvStr("SERVER_PORT")

	// store
	databaseURL := readEnvStr("DATABASE_URL")
	natsURL := readEnvStr("NATS_URL")

	// auth
	enableAuth := readEnvBool("ENABLE_AUTH")
	googleClientID := readEnvStr("GOOGLE_CLIENT_ID")
	googleClientSecret := readEnvStr("GOOGLE_CLIENT_SECRET")
	cookieHashKey := readEnvStr("COOKIE_HASH_KEY")
	cookieEncryptionKey := readEnvStr("COOKIE_ENCRYPTION_KEY")
	jwtEncodedPrivateKey := readEnvStr("JWT_PRIVATE_KEY")
	jwtEncodedPublicKey := readEnvStr("JWT_PUBLIC_KEY")

	// env
	dev := readEnvBool("DEV")

	priv, pub, err := decodeKeyPair([]byte(jwtEncodedPrivateKey), []byte(jwtEncodedPublicKey))
	if err != nil {
		log.Fatal(err)
	}

	return &Config{
		Server: ServerConfig{
			Port: serverPort,
		},
		Store: StoreConfig{
			DatabaseURL: databaseURL,
			NatsURL:     natsURL,
		},
		Auth: AuthConfig{
			Enable: enableAuth,
			Providers: ProvidersConfig{
				Google: GoogleConfig{
					ClientID:     googleClientID,
					ClientSecret: googleClientSecret,
				},
			},
			Keys: Keys{
				CookieHashKey:       cookieHashKey,
				CookieEncryptionKey: cookieEncryptionKey,
				JWTPrivateKey:       priv,
				JWTPublicKey:        pub,
			},
		},
		Dev: dev,
	}
}

func readEnvStr(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("rmx: no value assigned for key [%s]", key)
	}
	return v
}

/*
func readEnvInt(key string) (int, error) {
	s, err := readEnvStr(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}
*/

func readEnvBool(key string) bool {
	s := readEnvStr(key)
	v, err := strconv.ParseBool(s)
	if err != nil {
		log.Fatalf("rmx: couldn't parse (bool) value from key [%s]", key)
	}
	return v
}

// check for a config file
func decodeKeyPair(privEncoded, pubEncoded []byte) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	blockPriv, _ := pem.Decode(privEncoded)
	privX509Encoded := blockPriv.Bytes
	priv, err := x509.ParseECPrivateKey(privX509Encoded)
	if err != nil {
		return nil, nil, err
	}

	blockPub, _ := pem.Decode(pubEncoded)
	pubX509Encoded := blockPub.Bytes
	genericPubKey, err := x509.ParsePKIXPublicKey(pubX509Encoded)
	if err != nil {
		return nil, nil, err
	}
	pub, ok := genericPubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("public key not of type ecdsa.PublicKey")
	}

	return priv, pub, nil
}
