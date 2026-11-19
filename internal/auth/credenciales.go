// Credenciales por correo (14.5, espejo de ServicioCredenciales de auth.py).
// Repositorio inyectable: tabla `credenciales` de Postgres en produccion,
// memoria en fallback/tests. Siembra perezosa con la contrasena demo en el
// primer login; el cambio exige la actual y la politica minima de longitud.
package auth

import (
	"sync"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// RepositorioCredenciales: contrato comun de memoria y Postgres.
type RepositorioCredenciales interface {
	Listar() map[string]string
	Guardar(correo, hash string) error
	Eliminar(correo string) error
	EliminarExcepto(correos []string) error
}

// RepositorioCredencialesMemoria: fallback sin DATABASE_URL y tests.
type RepositorioCredencialesMemoria struct {
	mu     sync.Mutex
	hashes map[string]string
}

func NuevoCredencialesMemoria() *RepositorioCredencialesMemoria {
	return &RepositorioCredencialesMemoria{hashes: map[string]string{}}
}

func (r *RepositorioCredencialesMemoria) Listar() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.hashes))
	for k, v := range r.hashes {
		out[k] = v
	}
	return out
}

func (r *RepositorioCredencialesMemoria) Guardar(correo, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hashes[correo] = hash
	return nil
}

func (r *RepositorioCredencialesMemoria) Eliminar(correo string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hashes, correo)
	return nil
}

func (r *RepositorioCredencialesMemoria) EliminarExcepto(correos []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	conservar := map[string]bool{}
	for _, c := range correos {
		conservar[c] = true
	}
	for correo := range r.hashes {
		if !conservar[correo] {
			delete(r.hashes, correo)
		}
	}
	return nil
}

var _ RepositorioCredenciales = (*RepositorioCredencialesMemoria)(nil)

// ServicioCredenciales: hashes correo -> PBKDF2, sembrados de forma perezosa.
type ServicioCredenciales struct {
	repo RepositorioCredenciales
}

func NuevoCredenciales(repo RepositorioCredenciales) *ServicioCredenciales {
	if repo == nil {
		repo = NuevoCredencialesMemoria()
	}
	return &ServicioCredenciales{repo: repo}
}

// Autenticar: siembra la demo si no existe credencial y verifica.
func (s *ServicioCredenciales) Autenticar(correo, contrasena string) bool {
	almacenado, ok := s.repo.Listar()[correo]
	if !ok {
		almacenado = HashClave(ClaveDemo(), nil)
		_ = s.repo.Guardar(correo, almacenado)
	}
	return VerificarClave(contrasena, almacenado)
}

// Registrar: credencial inicial de un usuario nuevo (sobrescribe si re-crea).
func (s *ServicioCredenciales) Registrar(correo, contrasena string) {
	_ = s.repo.Guardar(correo, HashClave(contrasena, nil))
}

// Cambiar: exige la actual (re-verificacion) y la politica minima; false si
// la credencial no existe o la actual no verifica; ErrorValidacion si la
// nueva viola la politica.
func (s *ServicioCredenciales) Cambiar(correo, contrasenaActual, contrasenaNueva string) (bool, error) {
	almacenado, ok := s.repo.Listar()[correo]
	if !ok {
		almacenado = HashClave(ClaveDemo(), nil)
		_ = s.repo.Guardar(correo, almacenado)
	}
	if !VerificarClave(contrasenaActual, almacenado) {
		return false, nil
	}
	if !ClaveValida(contrasenaNueva) {
		return false, domain.ErrorValidacion("La contrasena nueva debe tener al menos %d caracteres.", LongitudMinimaClave)
	}
	_ = s.repo.Guardar(correo, HashClave(contrasenaNueva, nil))
	return true, nil
}

// Renombrar: mueve la credencial si el correo del usuario cambia.
func (s *ServicioCredenciales) Renombrar(correoAnterior, correoNuevo string) {
	if correoAnterior == correoNuevo {
		return
	}
	hashes := s.repo.Listar()
	hashActual, ok := hashes[correoAnterior]
	if !ok {
		return
	}
	_ = s.repo.Guardar(correoNuevo, hashActual)
	_ = s.repo.Eliminar(correoAnterior)
}

// SincronizarConUsuarios: limpia huerfanas tras un re-sync total (14.5).
// Firma compatible con servicio.CambioCredenciales.
func (s *ServicioCredenciales) SincronizarConUsuarios(correos map[string]bool) {
	lista := make([]string, 0, len(correos))
	for c := range correos {
		lista = append(lista, c)
	}
	_ = s.repo.EliminarExcepto(lista)
}

var _ interface {
	Renombrar(string, string)
	SincronizarConUsuarios(map[string]bool)
} = (*ServicioCredenciales)(nil)
