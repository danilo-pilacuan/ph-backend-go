package auth

import (
	"strings"
	"testing"
	"time"
)

// PBKDF2: paridad de formato con hash_contrasena de app/auth.py. El mismo
// hash debe verificar en ambos backends (tabla credenciales compartida).

func TestHashClaveFormatoCompatible(t *testing.T) {
	sal := make([]byte, 16)
	for i := range sal {
		sal[i] = byte(i)
	}
	hash := HashClave("projecthub-demo", sal)
	partes := strings.Split(hash, "$")
	if len(partes) != 4 {
		t.Fatalf("formato: %s", hash)
	}
	if partes[0] != "pbkdf2_sha256" || partes[1] != "240000" {
		t.Fatalf("algoritmo/iteraciones: %s", hash)
	}
	if partes[2] != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("sal hex: %s", partes[2])
	}
	// Verificacion con el hash generado.
	if !VerificarClave("projecthub-demo", hash) {
		t.Fatal("el hash debe verificar con la misma contrasena")
	}
	if VerificarClave("otra-clave", hash) {
		t.Fatal("el hash no debe verificar con otra contrasena")
	}
}

func TestHashClaveAleatorio(t *testing.T) {
	h1 := HashClave("projecthub-demo", nil)
	h2 := HashClave("projecthub-demo", nil)
	if h1 == h2 {
		t.Fatal("sales aleatorias deben producir hashes distintos")
	}
	if !VerificarClave("projecthub-demo", h1) || !VerificarClave("projecthub-demo", h2) {
		t.Fatal("ambos hashes deben verificar")
	}
}

func TestVerificarClaveFormatoInvalido(t *testing.T) {
	casos := []string{
		"",
		"texto-sin-formato",
		"md5$100$aa$bb",
		"pbkdf2_sha256$xxx$aa$bb",  // iteraciones no numericas
		"pbkdf2_sha256$100$zz$bb",  // sal no hex
		"pbkdf2_sha256$100$aa",     // faltan partes
		"pbkdf2_sha256$100$aa$gg",  // hash no hex
	}
	for _, caso := range casos {
		if VerificarClave("x", caso) {
			t.Fatalf("formato invalido debio rechazarse: %q", caso)
		}
	}
}

func TestClaveValidaYClaveDemo(t *testing.T) {
	if ClaveValida("corta") {
		t.Fatal("menos de 8 caracteres debe rechazarse")
	}
	if !ClaveValida("8caracteres") {
		t.Fatal("8 caracteres deben pasar")
	}
	if ClaveDemo() == "" {
		t.Fatal("clave demo no puede ser vacia")
	}
}

// JWT HS256: paridad con PyJWT. Un token generado por Python (formato
// header.payload.signature base64url) debe verificar aqui y viceversa.

func TestTokenFormatoYPayload(t *testing.T) {
	// Emitido ahora (fecha simulada del entorno) con exp a 7 dias: valido.
	token := CrearToken("u-admin", "secreto-de-test", nil)
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		t.Fatalf("token: %s", token)
	}
	// Header JSON canonico del mismo formato que PyJWT.
	if partes[0] != base64urlJSON(`{"alg":"HS256","typ":"JWT"}`) {
		t.Fatalf("header: %s", partes[0])
	}
	if LeerToken(token, "secreto-de-test") != "u-admin" {
		t.Fatal("token valido debe resolver el sub")
	}
	if LeerToken(token, "otro-secreto") != "" {
		t.Fatal("firma con otro secreto debe fallar")
	}
}

func TestTokenExpirado(t *testing.T) {
	// Emitido hace mas de 7 dias (exp en el pasado respecto de ahora).
	haceOchoDias := time.Now().Add(-8 * 24 * time.Hour)
	token := CrearToken("u-admin", "secreto-de-test", &haceOchoDias)
	if LeerToken(token, "secreto-de-test") != "" {
		t.Fatal("token con exp en el pasado debe fallar")
	}
}

func TestTokenFormatosInvalidos(t *testing.T) {
	casos := []string{"", "a.b", "a.b.c.d", "!!!.###.$$$"}
	for _, caso := range casos {
		if LeerToken(caso, "secreto") != "" {
			t.Fatalf("token invalido debio fallar: %q", caso)
		}
	}
}

func TestExtraerBearer(t *testing.T) {
	if ExtraerBearer("Bearer abc.def.ghi") != "abc.def.ghi" {
		t.Fatal("Bearer estandar debe extraer el token")
	}
	if ExtraerBearer("bearer abc") != "abc" {
		t.Fatal("esquema case-insensitive")
	}
	if ExtraerBearer("Bearer   ") != "" {
		t.Fatal("Bearer vacio debe dar vacio")
	}
	if ExtraerBearer("Basic abc") != "" {
		t.Fatal("otro esquema debe dar vacio")
	}
	if ExtraerBearer("") != "" {
		t.Fatal("header vacio debe dar vacio")
	}
}

func base64urlJSON(json string) string {
	// El encoder RawURLEncoding sin padding, igual que PyJWT.
	return b64url([]byte(json))
}
