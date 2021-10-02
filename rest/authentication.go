package rest

import (
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Sample code from https://betterprogramming.pub/hands-on-with-jwt-in-golang-8c986d1bb4c0

type LoginBody struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// JwtWrapper wraps the signing key and the issuer
type JWTWrapper struct {
	secretKey              string
	issuer                 string
	jwtExpirationMinutes   int64
	refreshExpirationHours int64
}

func NewJWTWrapper(key, issuer string, expiration, refreshExpiration int64) JWTWrapper {
	return JWTWrapper{
		secretKey:              key,
		issuer:                 issuer,
		jwtExpirationMinutes:   expiration,
		refreshExpirationHours: refreshExpiration,
	}
}

// JwtClaim adds name as a claim to the token
type JwtClaim struct {
	Name string
	jwt.StandardClaims
}

type JWTUser struct {
	Name string
}

// GenerateLoginToken generates a jwt token
func (wrap *JWTWrapper) GenerateLoginToken(name string) (signedToken string, err error) {
	claims := &JwtClaim{
		Name: name,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Minute * time.Duration(wrap.jwtExpirationMinutes)).Unix(),
			Issuer:    wrap.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err = token.SignedString([]byte(wrap.secretKey))
	if err != nil {
		return
	}

	return
}

// GenerateRefreshToken generates a jwt token intended to be used with a refresh cookie
func (wrap *JWTWrapper) GenerateRefreshToken(name string) (signedToken string, err error) {
	claims := &JwtClaim{
		Name: name,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(wrap.refreshExpirationHours)).Unix(),
			Issuer:    wrap.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err = token.SignedString([]byte(wrap.secretKey))
	if err != nil {
		return
	}

	return
}

//ValidateToken validates the jwt token
func (wrap *JWTWrapper) ValidateToken(signedToken string) (user JWTUser, err error) {
	var claims *JwtClaim
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JwtClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(wrap.secretKey), nil
		},
	)

	if err != nil {
		return
	}

	claims, ok := token.Claims.(*JwtClaim)
	if !ok {
		err = errors.New("could not parse claims")
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("token is expired")
		return
	}

	user.Name = claims.Name

	return

}

func (wrap *JWTWrapper) UserPassToToken(user, password string) (string, error) {

	// Placeholder credential checking. Very secure
	if password != "some password" || user != "kaese" {
		return "", errors.New("invalid user credentials")
	}

	signedToken, err := wrap.GenerateLoginToken(user)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (wrap *JWTWrapper) NewRefreshToken(user string) (string, error) {
	signedToken, err := wrap.GenerateRefreshToken(user)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
