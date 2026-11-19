package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/auth"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/servicio"
)

func nuevoServidorPrueba(t *testing.T) (*Servidor, http.Handler) {
	t.Helper()
	repo := almacen.NuevoMemoria(true)
	serv := servicio.NuevoServicio(repo)
	cred := auth.NuevoCredenciales(nil)
	serv.AsignarCredenciales(cred)
	s := NuevoServidor(serv, cred, "secreto-de-test")
	return s, s.Handler()
}

func usuarioAdmin(t *testing.T, s *Servidor) domain.Usuario {
	t.Helper()
	for _, u := range s.Servicio.Usuarios() {
		for _, r := range u.Roles {
			if r == domain.RolAdmin {
				return u
			}
		}
	}
	t.Fatal("no hay Admin en la semilla")
	return domain.Usuario{}
}

func usuarioPM(t *testing.T, s *Servidor) domain.Usuario {
	t.Helper()
	for _, u := range s.Servicio.Usuarios() {
		for _, r := range u.Roles {
			if r == domain.RolProjectManager {
				return u
			}
		}
	}
	t.Fatal("no hay PM en la semilla")
	return domain.Usuario{}
}

func peticionJSON(t *testing.T, handler http.Handler, metodo, ruta string, actor *domain.Usuario, cuerpo any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var lector *bytes.Reader
	if cuerpo != nil {
		brutos, _ := json.Marshal(cuerpo)
		lector = bytes.NewReader(brutos)
	} else {
		lector = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(metodo, ruta, lector)
	req.Header.Set("Content-Type", "application/json")
	if actor != nil {
		req.Header.Set("X-Actor-Id", actor.ID)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var salida map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &salida)
	}
	return rec, salida
}

func TestSaludPublica(t *testing.T) {
	_, handler := nuevoServidorPrueba(t)
	rec, cuerpo := peticionJSON(t, handler, "GET", "/api/salud", nil, nil)
	if rec.Code != 200 || cuerpo["estado"] != "ok" {
		t.Fatalf("salud: %d %v", rec.Code, cuerpo)
	}
}

func TestRutasProtegidasExigenActor(t *testing.T) {
	_, handler := nuevoServidorPrueba(t)
	rec, cuerpo := peticionJSON(t, handler, "GET", "/api/proyectos", nil, nil)
	// Sin credenciales: VALIDACION -> 422 (mismo contrato que rutas.py).
	if rec.Code != 422 || cuerpo["codigo"] != "VALIDACION" {
		t.Fatalf("sin actor: %d %v", rec.Code, cuerpo)
	}
}

func TestActorPorXActorIdY404DeUsuarioFantasma(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	rec, _ := peticionJSON(t, handler, "GET", "/api/proyectos", &admin, nil)
	if rec.Code != 200 {
		t.Fatalf("con actor: %d %s", rec.Code, rec.Body.String())
	}
	rec, cuerpo := peticionJSON(t, handler, "GET", "/api/proyectos", &domain.Usuario{ID: "u-fantasma"}, nil)
	if rec.Code != 404 || cuerpo["codigo"] != "ENTIDAD_NO_ENCONTRADA" {
		t.Fatalf("actor fantasma: %d %v", rec.Code, cuerpo)
	}
}

func TestLoginFlujoCompletoConJWT(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	// Credencial demo: registrar para el Admin de semilla.
	admin := usuarioAdmin(t, s)
	s.Credenciales.Registrar(admin.Correo, "projecthub-demo")

	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
		"correo": admin.Correo, "contrasena": "projecthub-demo",
	})
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	token, _ := cuerpo["token"].(string)
	if token == "" || cuerpo["tipo"] != "Bearer" {
		t.Fatalf("respuesta login: %v", cuerpo)
	}
	// El token resuelve al actor en /api/auth/me.
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != 200 {
		t.Fatalf("me con token: %d %s", rec2.Code, rec2.Body.String())
	}
	var me domain.Usuario
	_ = json.Unmarshal(rec2.Body.Bytes(), &me)
	if me.ID != admin.ID {
		t.Fatalf("me = %v", me)
	}
	// Token invalido: 401 SESION_INVALIDA.
	req3 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req3.Header.Set("Authorization", "Bearer no-es-un-token")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != 401 {
		t.Fatalf("token invalido: %d %s", rec3.Code, rec3.Body.String())
	}
}

func TestLoginCredencialesInvalidas401(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	s.Credenciales.Registrar(admin.Correo, "projecthub-demo")
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
		"correo": admin.Correo, "contrasena": "mala-clave",
	})
	if rec.Code != 401 || cuerpo["codigo"] != "CREDENCIALES_INVALIDAS" {
		t.Fatalf("login malo: %d %v", rec.Code, cuerpo)
	}
}

func TestRateLimitLogin429(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	s.Credenciales.Registrar(admin.Correo, "projecthub-demo")
	s.Limitador.intentos = 3
	s.Limitador.ventana = time.Minute
	for i := 0; i < 3; i++ {
		rec, _ := peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
			"correo": admin.Correo, "contrasena": "mala",
		})
		if rec.Code != 401 {
			t.Fatalf("fallo %d: %d", i, rec.Code)
		}
	}
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
		"correo": admin.Correo, "contrasena": "projecthub-demo",
	})
	if rec.Code != 429 || cuerpo["codigo"] != "LIMITE_INTENTOS" {
		t.Fatalf("debio bloquear: %d %v", rec.Code, cuerpo)
	}
}

func TestCRUDProyectosPorAPI(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	pm := usuarioPM(t, s)
	nombre := "Proyecto desde API"
	rec, creado := peticionJSON(t, handler, "POST", "/api/proyectos", &pm, map[string]any{
		"nombre": nombre, "presupuestoEstimado": 12345.0,
	})
	if rec.Code != 201 {
		t.Fatalf("crear: %d %s", rec.Code, rec.Body.String())
	}
	id, _ := creado["id"].(string)
	if id == "" || creado["etapa"] != "Planteamiento" || creado["estado"] != "Activo" {
		t.Fatalf("creado: %v", creado)
	}
	// Duplicado case-insensitive -> 409 DUPLICADO.
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/proyectos", &pm, map[string]any{"nombre": "PROYECTO DESDE API"})
	if rec.Code != 409 || cuerpo["codigo"] != "DUPLICADO" {
		t.Fatalf("duplicado: %d %v", rec.Code, cuerpo)
	}
	// Obtener por id y resumen.
	rec, _ = peticionJSON(t, handler, "GET", "/api/proyectos/"+id, &pm, nil)
	if rec.Code != 200 {
		t.Fatalf("obtener: %d", rec.Code)
	}
	rec, resumen := peticionJSON(t, handler, "GET", "/api/proyectos/"+id+"/resumen", &pm, nil)
	if rec.Code != 200 || resumen["presupuestoTotal"] != 0.0 {
		t.Fatalf("resumen: %d %v", rec.Code, resumen)
	}
	// El estimado del proyecto se conserva en el objeto proyecto anidado.
	if proyecto, ok := resumen["proyecto"].(map[string]any); !ok || proyecto["presupuestoEstimado"] != 12345.0 {
		t.Fatalf("resumen.proyecto: %v", resumen["proyecto"])
	}
	// 404 para proyecto inexistente.
	rec, cuerpo = peticionJSON(t, handler, "GET", "/api/proyectos/p-fantasma", &pm, nil)
	if rec.Code != 404 || cuerpo["codigo"] != "ENTIDAD_NO_ENCONTRADA" {
		t.Fatalf("fantasma: %d %v", rec.Code, cuerpo)
	}
	// Analista no puede crear -> 409 PERMISO_DENEGADO.
	analista := domain.Usuario{}
	for _, u := range s.Servicio.Usuarios() {
		if len(u.Roles) > 0 && u.Roles[0] == domain.RolAnalista {
			analista = u
		}
	}
	rec, cuerpo = peticionJSON(t, handler, "POST", "/api/proyectos", &analista, map[string]any{"nombre": "x"})
	if rec.Code != 409 || cuerpo["codigo"] != "PERMISO_DENEGADO" {
		t.Fatalf("permiso: %d %v", rec.Code, cuerpo)
	}
}

func TestEtapasYAvancePorAPI(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	pm := usuarioPM(t, s)
	rec, etapa := peticionJSON(t, handler, "POST", "/api/proyectos/p1/etapas", &pm, map[string]any{
		"nombre": "Etapa API", "fechaInicio": "2026-01-01", "fechaFin": "2026-01-31",
		"responsableId": pm.ID, "presupuestoAsignado": 1000.0,
	})
	if rec.Code != 201 {
		t.Fatalf("crear etapa: %d %s", rec.Code, rec.Body.String())
	}
	etapaID, _ := etapa["id"].(string)
	// Avance fuera de rango -> 422.
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/etapas/"+etapaID+"/avance", &pm, map[string]float64{"avance": 150})
	if rec.Code != 422 || cuerpo["codigo"] != "VALIDACION" {
		t.Fatalf("avance 150: %d %v", rec.Code, cuerpo)
	}
	// Avance valido.
	rec, _ = peticionJSON(t, handler, "POST", "/api/etapas/"+etapaID+"/avance", &pm, map[string]float64{"avance": 40})
	if rec.Code != 200 {
		t.Fatalf("avance 40: %d %s", rec.Code, rec.Body.String())
	}
	// Avance en e2 (depende de e1 al 100%) permitido; en e5 (e4 al 10%) 422.
	rec, cuerpo = peticionJSON(t, handler, "POST", "/api/etapas/e5/avance", &pm, map[string]float64{"avance": 10})
	if rec.Code != 422 {
		t.Fatalf("e5 con prerequisito incompleto: %d %v", rec.Code, cuerpo)
	}
}

func TestGateYTransicionPorAPI(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	rec, gate := peticionJSON(t, handler, "GET", "/api/proyectos/p2/gate", &admin, nil)
	if rec.Code != 200 || gate["abierto"] != false {
		t.Fatalf("gate p2: %d %v", rec.Code, gate)
	}
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/proyectos/p2/transicionar", &admin, map[string]any{
		"destino": "Contratos",
	})
	if rec.Code != 422 || cuerpo["codigo"] != "VALIDACION" {
		t.Fatalf("transicionar con gate cerrado: %d %v", rec.Code, cuerpo)
	}
}

func TestSincronizacionGETyPUT(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	rec, estado := peticionJSON(t, handler, "GET", "/api/sincronizacion", &admin, nil)
	if rec.Code != 200 {
		t.Fatalf("GET sincronizacion: %d", rec.Code)
	}
	usuarios, _ := estado["usuarios"].([]any)
	if len(usuarios) != 7 {
		t.Fatalf("usuarios = %d", len(usuarios))
	}
	// PUT del mismo estado: roundtrip sin cambios.
	rec, _ = peticionJSON(t, handler, "PUT", "/api/sincronizacion", &admin, estado)
	if rec.Code != 200 {
		t.Fatalf("PUT sincronizacion: %d %s", rec.Code, rec.Body.String())
	}
	// PUT sin Admin: PERMISO_DENEGADO.
	pm := usuarioPM(t, s)
	rec, cuerpo := peticionJSON(t, handler, "PUT", "/api/sincronizacion", &pm, estado)
	if rec.Code != 409 || cuerpo["codigo"] != "PERMISO_DENEGADO" {
		t.Fatalf("PUT sin admin: %d %v", rec.Code, cuerpo)
	}
}

func TestCorsSoloOrigenesPermitidos(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	// Origen permitido: se ecoa.
	req := httptest.NewRequest("GET", "/api/salud", nil)
	req.Header.Set("Origin", "http://localhost:4300")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4300" {
		t.Fatalf("CORS permitido: %v", rec.Header())
	}
	// Origen desconocido: sin cabecera CORS.
	req2 := httptest.NewRequest("GET", "/api/salud", nil)
	req2.Header.Set("Origin", "http://malo.example")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("CORS rechazado debia omitirse: %v", rec2.Header())
	}
	// Preflight OPTIONS: 204 con cabeceras.
	req3 := httptest.NewRequest("OPTIONS", "/api/proyectos", nil)
	req3.Header.Set("Origin", "http://localhost:4300")
	req3.Header.Set("Access-Control-Request-Method", "POST")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != 204 {
		t.Fatalf("preflight: %d", rec3.Code)
	}
	_ = s
}

func TestCambioContrasenaPorAPI(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	s.Credenciales.Registrar(admin.Correo, "projecthub-demo")
	// Actual incorrecta -> 401 CONTRASENA_ACTUAL_INVALIDA.
	rec, cuerpo := peticionJSON(t, handler, "PUT", "/api/auth/contrasena", &admin, map[string]string{
		"contrasenaActual": "mala", "contrasenaNueva": "nueva-clave-1",
	})
	if rec.Code != 401 || cuerpo["codigo"] != "CONTRASENA_ACTUAL_INVALIDA" {
		t.Fatalf("cambio con actual mala: %d %v", rec.Code, cuerpo)
	}
	// Nueva sin politica -> 422 VALIDACION.
	rec, cuerpo = peticionJSON(t, handler, "PUT", "/api/auth/contrasena", &admin, map[string]string{
		"contrasenaActual": "projecthub-demo", "contrasenaNueva": "corta",
	})
	if rec.Code != 422 || cuerpo["codigo"] != "VALIDACION" {
		t.Fatalf("cambio con nueva corta: %d %v", rec.Code, cuerpo)
	}
	// Cambio correcto.
	rec, cuerpo = peticionJSON(t, handler, "PUT", "/api/auth/contrasena", &admin, map[string]string{
		"contrasenaActual": "projecthub-demo", "contrasenaNueva": "nueva-clave-1",
	})
	if rec.Code != 200 || cuerpo["cambiada"] != true {
		t.Fatalf("cambio ok: %d %v", rec.Code, cuerpo)
	}
	// El login viejo ya no funciona; el nuevo si.
	rec, _ = peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
		"correo": admin.Correo, "contrasena": "projecthub-demo",
	})
	if rec.Code != 401 {
		t.Fatalf("login viejo debio fallar: %d", rec.Code)
	}
	rec, _ = peticionJSON(t, handler, "POST", "/api/auth/login", nil, map[string]string{
		"correo": admin.Correo, "contrasena": "nueva-clave-1",
	})
	if rec.Code != 200 {
		t.Fatalf("login nuevo debio pasar: %d %s", rec.Code, rec.Body.String())
	}
}

func TestEntregablesYAprobacionPorAPI(t *testing.T) {
	s, handler := nuevoServidorPrueba(t)
	admin := usuarioAdmin(t, s)
	pm := usuarioPM(t, s)
	// El PM crea el entregable (el autor no puede autoaprobarse).
	rec, entregable := peticionJSON(t, handler, "POST", "/api/proyectos/p1/entregables", &pm, map[string]any{
		"etapa": "Ejecucion", "nombre": "Acta de avance de ejecucion",
	})
	if rec.Code != 201 {
		t.Fatalf("crear entregable: %d %s", rec.Code, rec.Body.String())
	}
	id, _ := entregable["id"].(string)
	// El autor (PM) ni siquiera tiene permiso de aprobar -> 409 PERMISO_DENEGADO.
	rec, cuerpo := peticionJSON(t, handler, "POST", "/api/entregables/"+id+"/aprobar", &pm, nil)
	if rec.Code != 409 || cuerpo["codigo"] != "PERMISO_DENEGADO" {
		t.Fatalf("aprobacion por PM: %d %v", rec.Code, cuerpo)
	}
	// Un Admin (gobernanza) si puede.
	rec, _ = peticionJSON(t, handler, "POST", "/api/entregables/"+id+"/aprobar", &admin, nil)
	if rec.Code != 200 {
		t.Fatalf("aprobacion admin: %d %s", rec.Code, rec.Body.String())
	}
	// No se reaprueba -> 422.
	rec, _ = peticionJSON(t, handler, "POST", "/api/entregables/"+id+"/aprobar", &admin, nil)
	if rec.Code != 422 {
		t.Fatalf("reaprobacion debio fallar: %d", rec.Code)
	}
}
