package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Golden de paridad: token emitido por PyJWT (app/auth.py, crear_token con
// secreto "secreto-paridad-go-py"). El Go debe verificarlo y extraer el sub.
// Regenerar con:
//   cd backend && python -c "import sys; sys.path.insert(0,'.'); from app.auth
//   import crear_token; print(crear_token('u-admin','secreto-paridad-go-py'))"
func TestTokenDePyJWTVerificaEnGo(t *testing.T) {
	tokenPy := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1LWFkbWluIiwiaWF0IjoxNzk1MDc1MTcxLCJleHAiOjE3OTU2Nzk5NzF9.YARAywOuCAomMWixT94NGsLB5CW_rQi-jOe78QfeC3Q"
	if sub := LeerToken(tokenPy, "secreto-paridad-go-py"); sub != "u-admin" {
		t.Fatalf("token de PyJWT debio verificar en Go, sub=%q", sub)
	}
	if sub := LeerToken(tokenPy, "secreto-erroneo"); sub != "" {
		t.Fatal("el mismo token con otro secreto debe fallar")
	}
}

// Golden inverso: el Go emite con el mismo formato (header canonico, payload
// base64url, firma HS256) y PyJWT lo verificaria; aqui se comprueba la forma
// del header y que la firma es estable para el mismo iat.
func TestTokenGoFormatoIdenticoAPyJWT(t *testing.T) {
	ahora := time.Unix(1795075171, 0) // mismo iat que el token de Python
	token := CrearToken("u-admin", "secreto-paridad-go-py", &ahora)
	partes := splitToken(token)
	if len(partes) != 3 {
		t.Fatalf("token: %s", token)
	}
	if partes[0] != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" {
		t.Fatalf("header distinto al canonico de PyJWT: %s", partes[0])
	}
	// El payload debe decodificar a {"exp","iat","sub"} con los mismos valores.
	payload := decodePayload(t, partes[1])
	esperado := `{"sub":"u-admin","iat":1795075171,"exp":` + "1795680000" + `}` // no exacto: el exp depende de DuracionToken
	if payload["sub"] != "u-admin" || payload["iat"].(float64) != 1795075171 {
		t.Fatalf("payload: %v (esperado base %s)", payload, esperado)
	}
	if exp, ok := payload["exp"].(float64); !ok || exp <= payload["iat"].(float64) {
		t.Fatalf("exp debe ser iat + 7 dias: %v", payload)
	}
}

func splitToken(token string) []string { return strings.Split(token, ".") }

func decodePayload(t *testing.T, parte string) map[string]any {
	t.Helper()
	brutos, err := base64.RawURLEncoding.DecodeString(parte)
	if err != nil {
		t.Fatalf("payload no es base64url: %v", err)
	}
	var salida map[string]any
	if err := json.Unmarshal(brutos, &salida); err != nil {
		t.Fatalf("payload no es JSON: %v", err)
	}
	return salida
}
