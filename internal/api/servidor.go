// Package api: servidor HTTP (espejo de app/main.py + app/api/rutas.py).
// net/http con patrones Go 1.22+, CORS propio, actor por Bearer JWT o
// X-Actor-Id, rate limit de login y body de error {codigo, mensaje}.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/auth"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/servicio"
)

// Servidor: cableado de la API (servicio + credenciales + secreto).
type Servidor struct {
	Servicio     *servicio.ServicioProyectos
	Credenciales *auth.ServicioCredenciales
	Secreto      string
	Limitador    *LimitadorLogin
	Origenes     map[string]bool
}

type ServicioCredenciales = auth.ServicioCredenciales

// ---------- Errores HTTP ----------

type cuerpoError struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

func escribirError(w http.ResponseWriter, estado int, codigo, mensaje string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(cuerpoError{Codigo: codigo, Mensaje: mensaje})
}

// mapearErrorDominio: 404 no encontrado, 409 conflicto/permiso/estado/
// transicion, 422 validacion (mismos codigos que el handler de main.py).
func mapearErrorDominio(w http.ResponseWriter, err error) {
	de, ok := err.(*domain.ErrorDominio)
	if !ok {
		log.Printf("ERROR interno: %v", err)
		escribirError(w, http.StatusInternalServerError, "ERROR_INTERNO", "Error interno del servidor.")
		return
	}
	estado, ok := domain.CodigoHTTP[de.Codigo]
	if !ok {
		estado = http.StatusInternalServerError
	}
	escribirError(w, estado, string(de.Codigo), de.Mensaje)
}

func errorHTTP(estado int, codigo, mensaje string) *errorHTTPTipo { return &errorHTTPTipo{estado, codigo, mensaje} }

type errorHTTPTipo struct {
	estado  int
	codigo  string
	mensaje string
}

func (e *errorHTTPTipo) Error() string { return fmt.Sprintf("%d %s: %s", e.estado, e.codigo, e.mensaje) }

// ---------- Cuerpos de peticion (esquemas.py) ----------

type CuerpoProyecto struct {
	Nombre              string   `json:"nombre"`
	Descripcion         string   `json:"descripcion"`
	PatrocinadorID      *string  `json:"patrocinadorId"`
	PresupuestoEstimado *float64 `json:"presupuestoEstimado"`
	FechaObjetivo       *string  `json:"fechaObjetivo"`
}

type CuerpoEdicionProyecto struct {
	Nombre        *string `json:"nombre"`
	Descripcion   *string `json:"descripcion"`
	FechaObjetivo *string `json:"fechaObjetivo"`
}

type CuerpoComentario struct {
	Comentario *string `json:"comentario"`
}

type CuerpoLogin struct {
	Correo     string `json:"correo"`
	Contrasena string `json:"contrasena"`
}

type CuerpoCambioContrasena struct {
	ContrasenaActual string `json:"contrasenaActual"`
	ContrasenaNueva  string `json:"contrasenaNueva"`
}

type CuerpoTransicion struct {
	Destino    domain.EtapaCicloVida `json:"destino"`
	Comentario *string               `json:"comentario"`
}

type CuerpoEtapa struct {
	Nombre              string    `json:"nombre"`
	FechaInicio         string    `json:"fechaInicio"`
	FechaFin            string    `json:"fechaFin"`
	ResponsableID       string    `json:"responsableId"`
	PresupuestoAsignado *float64  `json:"presupuestoAsignado"`
	Avance              *float64  `json:"avance"`
	Dependencias        *[]string `json:"dependencias"`
}

type CuerpoEdicionEtapa struct {
	Nombre              *string   `json:"nombre"`
	FechaInicio         *string   `json:"fechaInicio"`
	FechaFin            *string   `json:"fechaFin"`
	ResponsableID       *string   `json:"responsableId"`
	PresupuestoAsignado *float64  `json:"presupuestoAsignado"`
	Avance              *float64  `json:"avance"`
	Dependencias        *[]string `json:"dependencias"`
}

type CuerpoAvance struct {
	Avance float64 `json:"avance"`
}

type CuerpoConsumo struct {
	Consumo float64 `json:"consumo"`
}

type CuerpoEntregable struct {
	Etapa   domain.EtapaCicloVida `json:"etapa"`
	Nombre  string                `json:"nombre"`
	Version *int                  `json:"version"`
}

type CuerpoUsuario struct {
	Nombre string        `json:"nombre"`
	Correo string        `json:"correo"`
	Roles  []domain.Rol  `json:"roles"`
	Activo *bool         `json:"activo"`
}

type CuerpoEdicionUsuario struct {
	Nombre *string       `json:"nombre"`
	Correo *string       `json:"correo"`
	Roles  *[]domain.Rol `json:"roles"`
	Activo *bool         `json:"activo"`
}

func decodificar[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		escribirError(w, http.StatusUnprocessableEntity, "VALIDACION", "El cuerpo de la peticion no es JSON valido.")
		return false
	}
	return true
}

func codificar(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------- Rate limit (rate_limit.py, VULN-003) ----------

type LimitadorLogin struct {
	mu       sync.Mutex
	fallos   map[string][]time.Time
	intentos int
	ventana  time.Duration
}

func NuevoLimitador() *LimitadorLogin {
	intentos := 5
	if v := os.Getenv("RATE_LIMIT_LOGIN_INTENTOS"); v != "" {
		if n, err := fmtSscan(v); err == nil {
			intentos = n
		}
	}
	ventanaSeg := 60
	if v := os.Getenv("RATE_LIMIT_LOGIN_VENTANA_SEG"); v != "" {
		if n, err := fmtSscan(v); err == nil {
			ventanaSeg = n
		}
	}
	return &LimitadorLogin{fallos: map[string][]time.Time{}, intentos: intentos, ventana: time.Duration(ventanaSeg) * time.Second}
}

func fmtSscan(v string) (int, error) {
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	return n, err
}

func (l *LimitadorLogin) registrarFallo(clave string, ahora time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lista := append(l.fallos[clave], ahora)
	recientes := lista[:0]
	for _, t := range lista {
		if t.After(ahora.Add(-l.ventana)) {
			recientes = append(recientes, t)
		}
	}
	l.fallos[clave] = recientes
}

func (l *LimitadorLogin) bloqueado(clave string, ahora time.Time) bool {
	if l.intentos <= 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	recientes := l.fallos[clave][:0]
	for _, t := range l.fallos[clave] {
		if t.After(ahora.Add(-l.ventana)) {
			recientes = append(recientes, t)
		}
	}
	l.fallos[clave] = recientes
	return len(recientes) >= l.intentos
}

func (l *LimitadorLogin) Limpiar() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fallos = map[string][]time.Time{}
}

// ---------- Actor (obtener_actor de rutas.py) ----------

func (s *Servidor) obtenerActor(w http.ResponseWriter, r *http.Request) (domain.Usuario, error) {
	if token := auth.ExtraerBearer(r.Header.Get("Authorization")); token != "" {
		usuarioID := auth.LeerToken(token, s.Secreto)
		if usuarioID == "" {
			return domain.Usuario{}, errorHTTP(401, "SESION_INVALIDA", "Tu sesion ha expirado; inicia sesion de nuevo.")
		}
		actor := s.Servicio.UsuarioPorID(usuarioID)
		if actor == nil {
			return domain.Usuario{}, errorHTTP(401, "SESION_INVALIDA", "Tu usuario ya no existe en el sistema; inicia sesion de nuevo.")
		}
		if !actor.Activo {
			return domain.Usuario{}, errorHTTP(403, "USUARIO_INACTIVO", "Tu cuenta fue desactivada; contacta a un administrador.")
		}
		return *actor, nil
	}
	xActor := r.Header.Get("X-Actor-Id")
	if xActor == "" {
		return domain.Usuario{}, domain.ErrorValidacion("Falta autenticar la sesion (Bearer token o X-Actor-Id).")
	}
	actor := s.Servicio.UsuarioPorID(xActor)
	if actor == nil {
		return domain.Usuario{}, domain.ErrorEntidadNoEncontrada("Usuario", xActor)
	}
	return *actor, nil
}

// ---------- Origenes CORS ----------

func origenesPermitidos() map[string]bool {
	bruta := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	origenes := map[string]bool{}
	if bruta != "" {
		for _, o := range strings.Split(bruta, ",") {
			o = strings.TrimRight(strings.TrimSpace(o), "/")
			if o != "" {
				origenes[o] = true
			}
		}
		if len(origenes) > 0 {
			return origenes
		}
	}
	for _, o := range []string{
		"http://localhost:4300", "http://localhost:4400",
		"http://localhost:4301", "http://localhost:4302",
		"http://localhost:4303", "http://localhost:4304",
	} {
		origenes[o] = true
	}
	return origenes
}

// corsMiddleware: origin eco solo si esta permitido; sin credenciales.
func (s *Servidor) corsMiddleware(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origen := r.Header.Get("Origin")
		if origen != "" && s.Origenes[origen] {
			w.Header().Set("Access-Control-Allow-Origin", origen)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Actor-Id, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		proximo.ServeHTTP(w, r)
	})
}

// NuevoServidor: cablea servicio, credenciales, secreto y limitador.
func NuevoServidor(s *servicio.ServicioProyectos, cred *ServicioCredenciales, secreto string) *Servidor {
	return &Servidor{
		Servicio:     s,
		Credenciales: cred,
		Secreto:      secreto,
		Limitador:    NuevoLimitador(),
		Origenes:     origenesPermitidos(),
	}
}

// Handler: todas las rutas bajo el middleware CORS.
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registrarRutas(mux)
	return s.corsMiddleware(mux)
}

var _ = almacen.NuevoMemoria // referencia para documentar la relacion con almacen
