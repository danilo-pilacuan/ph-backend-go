// Package auth: JWT HS256 sin dependencias externas (paridad con PyJWT en
// app/auth.py). Solo sub/iat/exp; HS256 es el unico algoritmo aceptado.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// CrearToken: JWT HS256 con sub y exp (7 dias), paridad con crear_token.
func CrearToken(usuarioID, secreto string, ahora *time.Time) string {
	emitido := time.Now()
	if ahora != nil {
		emitido = *ahora
	}
	claims := Claims{
		Sub: usuarioID,
		Iat: emitido.Unix(),
		Exp: emitido.Add(DuracionTokenSegundos * time.Second).Unix(),
	}
	header := b64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	firma := hmacSHA256([]byte(header+"."+string(payload)), []byte(secreto))
	return header + "." + string(payload) + "." + b64url(firma)
}

// LeerToken: id del usuario si el token es valido y no expiro; "" si no.
func LeerToken(token, secreto string) string {
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return ""
	}
	firma, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil {
		return ""
	}
	esperada := hmacSHA256([]byte(partes[0]+"."+partes[1]), []byte(secreto))
	if !hmac.Equal(firma, esperada) {
		return ""
	}
	var claims Claims
	if err := json.Unmarshal([]byte(partes[1]), &claims); err != nil {
		return ""
	}
	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		return ""
	}
	return claims.Sub
}

// ExtraerBearer: token de un header "Authorization: Bearer <jwt>"; "" si no aplica.
func ExtraerBearer(authorization string) string {
	partes := strings.SplitN(authorization, " ", 2)
	if len(partes) == 2 && strings.EqualFold(partes[0], "bearer") && strings.TrimSpace(partes[1]) != "" {
		return strings.TrimSpace(partes[1])
	}
	return ""
}

func hmacSHA256(datos, clave []byte) []byte {
	mac := hmac.New(sha256.New, clave)
	mac.Write(datos)
	return mac.Sum(nil)
}

// Ver errores de formato como errors compatibles.
var ErrTokenInvalido = errors.New("token invalido")

var _ = fmt.Sprintf // placeholder para evitar import vacio si se refactoriza
