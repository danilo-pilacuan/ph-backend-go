// Package servicio: usuarios (parte 4, espejo de app/servicio.py 1.1-1.8).
// Unicidad de correo case-insensitive, al menos un rol, y el sistema conserva
// siempre un Admin activo.
package servicio

import (
	"strings"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// DatosUsuario: entrada de crear/editar usuario.
type DatosUsuario struct {
	Nombre *string
	Correo *string
	Roles  *[]domain.Rol
	Activo *bool
}

var correoValido = domain.EsCorreoValido

func (s *ServicioProyectos) UsuariosActivos() []domain.Usuario {
	var out []domain.Usuario
	for _, u := range s.repo.Usuarios() {
		if u.Activo {
			out = append(out, u)
		}
	}
	return out
}

// ExigirAdmin: solo un Admin administra usuarios (1.8).
func (s *ServicioProyectos) ExigirAdmin(actor domain.Usuario) error {
	for _, r := range actor.Roles {
		if r == domain.RolAdmin {
			return nil
		}
	}
	return domain.ErrorPermisoDenegado("administrar usuarios", "sistema", []domain.Rol{domain.RolAdmin})
}

func validarNombre(nombre string) (string, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "", domain.ErrorValidacion("El nombre del usuario es obligatorio.")
	}
	return nombre, nil
}

func validarCorreo(correo string, exceptoID string, usuarios []domain.Usuario) (string, error) {
	correo = strings.ToLower(strings.TrimSpace(correo))
	if !correoValido(correo) {
		return "", domain.ErrorValidacion("El correo no tiene un formato valido.")
	}
	for _, u := range usuarios {
		if u.ID != exceptoID && u.Correo == correo {
			return "", domain.ErrorDuplicado("usuario", "correo", correo)
		}
	}
	return correo, nil
}

func validarRoles(roles []domain.Rol) ([]domain.Rol, error) {
	if len(roles) == 0 {
		return nil, domain.ErrorValidacion("El usuario requiere al menos un rol (1.4).")
	}
	validos := map[domain.Rol]bool{
		domain.RolAdmin: true, domain.RolSponsor: true, domain.RolProjectManager: true,
		domain.RolAnalista: true, domain.RolStakeholder: true,
	}
	for _, r := range roles {
		if !validos[r] {
			return nil, domain.ErrorValidacion("El rol %q no es un rol valido.", string(r))
		}
	}
	return roles, nil
}

// CrearUsuario: solo Admin; correo unico case-insensitive.
func (s *ServicioProyectos) CrearUsuario(datos DatosUsuario, actor domain.Usuario) (domain.Usuario, error) {
	if err := s.ExigirAdmin(actor); err != nil {
		return domain.Usuario{}, err
	}
	nombre, err := validarNombre(deref(datos.Nombre))
	if err != nil {
		return domain.Usuario{}, err
	}
	correo, err := validarCorreo(deref(datos.Correo), "", s.repo.Usuarios())
	if err != nil {
		return domain.Usuario{}, err
	}
	roles := []domain.Rol{}
	if datos.Roles != nil {
		roles = *datos.Roles
	}
	roles, err = validarRoles(roles)
	if err != nil {
		return domain.Usuario{}, err
	}
	activo := true
	if datos.Activo != nil {
		activo = *datos.Activo
	}
	usuario := domain.Usuario{
		ID:     NuevoID("u"),
		Nombre: nombre,
		Correo: correo,
		Roles:  roles,
		Activo: activo,
	}
	s.repo.UsuariosAgregar(usuario)
	return usuario, nil
}

// EditarUsuario: solo Admin; renombra credencial si cambia el correo; nunca
// deja al sistema sin su ultimo Admin activo.
func (s *ServicioProyectos) EditarUsuario(usuarioID string, datos DatosUsuario, actor domain.Usuario) (domain.Usuario, error) {
	if err := s.ExigirAdmin(actor); err != nil {
		return domain.Usuario{}, err
	}
	actual := s.UsuarioPorID(usuarioID)
	if actual == nil {
		return domain.Usuario{}, domain.ErrorValidacion("El usuario %q no existe.", usuarioID)
	}
	if datos.Nombre != nil {
		validado, err := validarNombre(*datos.Nombre)
		if err != nil {
			return domain.Usuario{}, err
		}
		actual.Nombre = validado
	}
	if datos.Correo != nil {
		validado, err := validarCorreo(*datos.Correo, usuarioID, s.repo.Usuarios())
		if err != nil {
			return domain.Usuario{}, err
		}
		if validado != actual.Correo && s.credCambio != nil {
			s.credCambio.Renombrar(actual.Correo, validado)
		}
		actual.Correo = validado
	}
	if datos.Roles != nil {
		roles, err := validarRoles(*datos.Roles)
		if err != nil {
			return domain.Usuario{}, err
		}
		actual.Roles = roles
	}
	if datos.Activo != nil {
		actual.Activo = *datos.Activo
	}
	if err := s.exigirUltimoAdminIntacto(usuarioID, actual.Roles, actual.Activo); err != nil {
		return domain.Usuario{}, err
	}
	lista := s.repo.Usuarios()
	for i := range lista {
		if lista[i].ID == usuarioID {
			lista[i] = *actual
			break
		}
	}
	s.repo.UsuariosReemplazar(lista)
	return *actual, nil
}

// DesactivarUsuario: azucar de EditarUsuario(activo=false).
func (s *ServicioProyectos) DesactivarUsuario(usuarioID string, actor domain.Usuario) (domain.Usuario, error) {
	falso := false
	return s.EditarUsuario(usuarioID, DatosUsuario{Activo: &falso}, actor)
}

// exigirUltimoAdminIntacto: el sistema conserva siempre un Admin activo
// (hallazgo de la iteracion 50).
func (s *ServicioProyectos) exigirUltimoAdminIntacto(usuarioID string, roles []domain.Rol, activo bool) error {
	for _, u := range s.repo.Usuarios() {
		if u.ID == usuarioID {
			continue
		}
		if u.Activo && tieneRol(u.Roles, domain.RolAdmin) {
			return nil
		}
	}
	if !(activo && tieneRol(roles, domain.RolAdmin)) {
		return domain.ErrorValidacion("El sistema debe conservar al menos un Admin activo; promueve a otro Admin antes de retirar el ultimo.")
	}
	return nil
}

func tieneRol(roles []domain.Rol, objetivo domain.Rol) bool {
	for _, r := range roles {
		if r == objetivo {
			return true
		}
	}
	return false
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
