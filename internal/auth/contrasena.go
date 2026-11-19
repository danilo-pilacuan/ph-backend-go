// Package auth: credenciales y token de sesion (espejo de app/auth.py).
//
// Diseno identico:
//   - Contrasena: PBKDF2-HMAC-SHA256 con sal aleatoria de 16 bytes y 240000
//     iteraciones; formato "pbkdf2_sha256$iteraciones$sal_hex$hash_hex".
//     Los hashes legados de la tabla `credenciales` de Postgres se verifican
//     sin re-hash: misma codificacion que hashlib.pbkdf2_hmac de Python.
//   - Token: JWT HS256 con sub = id de usuario, iat y exp (7 dias). La firma
//     vive en AUTH_SECRET (env); si no esta definida se genera una aleatoria
//     al arrancar (los tokens no sobreviven al reinicio) con warning en el log.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	Iteraciones           = 240000
	DuracionTokenSegundos = 7 * 24 * 3600
	LongitudMinimaClave   = 8
	ClaveDemoDefault      = "projecthub-demo"
)

// ClaveDemo: contrasena demo de semilla y usuarios nuevos (config, no literal).
func ClaveDemo() string {
	if v := strings.TrimSpace(os.Getenv("AUTH_CONTRASENA_DEMO")); v != "" {
		return v
	}
	return ClaveDemoDefault
}

// ClaveValida: politica minima de la contrasena nueva (14.5).
func ClaveValida(contrasena string) bool {
	return len(contrasena) >= LongitudMinimaClave
}

// HashClave: PBKDF2-HMAC-SHA256; formato compatible con hash_contrasena de Python.
func HashClave(contrasena string, sal []byte) string {
	if sal == nil {
		sal = make([]byte, 16)
		if _, err := rand.Read(sal); err != nil {
			panic(err)
		}
	}
	derivado := pbkdf2SHA256([]byte(contrasena), sal, Iteraciones, 32)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", Iteraciones, hex.EncodeToString(sal), hex.EncodeToString(derivado))
}

// VerificarClave: comparacion constante; false ante cualquier formato inesperado.
func VerificarClave(contrasena, almacenado string) bool {
	partes := strings.Split(almacenado, "$")
	if len(partes) != 4 || partes[0] != "pbkdf2_sha256" {
		return false
	}
	iteraciones, err := strconv.Atoi(partes[1])
	if err != nil || iteraciones <= 0 {
		return false
	}
	sal, err := hex.DecodeString(partes[2])
	if err != nil {
		return false
	}
	hashAlmacenado, err := hex.DecodeString(partes[3])
	if err != nil {
		return false
	}
	derivado := pbkdf2SHA256([]byte(contrasena), sal, iteraciones, len(hashAlmacenado))
	return subtle.ConstantTimeCompare(derivado, hashAlmacenado) == 1
}

// pbkdf2SHA256: PBKDF2 (RFC 2898) con HMAC-SHA256, sin dependencias externas.
func pbkdf2SHA256(contrasena, sal []byte, iteraciones, longitud int) []byte {
	prf := hmac.New(sha256.New, contrasena)
	hashLen := prf.Size()
	bloques := (longitud + hashLen - 1) / hashLen
	derivado := make([]byte, 0, bloques*hashLen)
	buf := make([]byte, 4)
	for bloque := 1; bloque <= bloques; bloque++ {
		prf.Reset()
		prf.Write(sal)
		binary.BigEndian.PutUint32(buf, uint32(bloque))
		prf.Write(buf)
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iteraciones; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		derivado = append(derivado, t...)
	}
	return derivado[:longitud]
}

// Secreto: AUTH_SECRET de entorno; sin ella, secreto efimero con warning.
func Secreto(logger *log.Logger) string {
	if s := strings.TrimSpace(os.Getenv("AUTH_SECRET")); s != "" {
		return s
	}
	logger.Printf("AVISO: AUTH_SECRET no definida: se genera un secreto efimero; los tokens no sobreviviran al reinicio del servicio.")
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
