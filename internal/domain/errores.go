// Package domain: errores de dominio tipados (espejo de domain/errores.py).
// El Codigo es estable y la API lo mapea a HTTP igual que la version Python.
package domain

import "fmt"

// CodigoError: codigos estables que la UI puede traducir.
type CodigoError string

const (
	CodigoTransicionInvalida    CodigoError = "TRANSICION_INVALIDA"
	CodigoPermisoDenegado       CodigoError = "PERMISO_DENEGADO"
	CodigoValidacion            CodigoError = "VALIDACION"
	CodigoEstadoProyecto        CodigoError = "ESTADO_PROYECTO"
	CodigoEntidadNoEncontrada   CodigoError = "ENTIDAD_NO_ENCONTRADA"
	CodigoDuplicado             CodigoError = "DUPLICADO"
	CodigoSesionInvalida        CodigoError = "SESION_INVALIDA"
	CodigoUsuarioInactivo       CodigoError = "USUARIO_INACTIVO"
	CodigoCredencialesInvalidas CodigoError = "CREDENCIALES_INVALIDAS"
	CodigoLimiteIntentos        CodigoError = "LIMITE_INTENTOS"
	CodigoContrasenaActualInv   CodigoError = "CONTRASENA_ACTUAL_INVALIDA"
)

// CodigoHTTP: mapeo codigo de dominio -> HTTP (10.3).
var CodigoHTTP = map[CodigoError]int{
	CodigoEntidadNoEncontrada: 404,
	CodigoDuplicado:           409,
	CodigoPermisoDenegado:     409,
	CodigoEstadoProyecto:      409,
	CodigoTransicionInvalida:  409,
	CodigoValidacion:          422,
}

// ErrorDominio: toda regla de negocio rechazada viaja en este tipo; la capa
// HTTP lo serializa como {"codigo": "...", "mensaje": "..."}.
type ErrorDominio struct {
	Codigo  CodigoError
	Mensaje string
}

func (e *ErrorDominio) Error() string { return fmt.Sprintf("%s: %s", e.Codigo, e.Mensaje) }

func NuevoError(codigo CodigoError, formato string, args ...any) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: fmt.Sprintf(formato, args...)}
}

func ErrorValidacion(formato string, args ...any) *ErrorDominio {
	return NuevoError(CodigoValidacion, formato, args...)
}

func ErrorEntidadNoEncontrada(entidad, id string) *ErrorDominio {
	return NuevoError(CodigoEntidadNoEncontrada, "%s con id %q no existe.", entidad, id)
}

func ErrorDuplicado(entidad, campo, valor string) *ErrorDominio {
	return NuevoError(CodigoDuplicado, "Ya existe %s con %s %q.", entidad, campo, valor)
}

func ErrorTransicionInvalida(origen, destino EtapaCicloVida) *ErrorDominio {
	return NuevoError(CodigoTransicionInvalida,
		"Transicion invalida: no se puede pasar de %q a %q.", string(origen), string(destino))
}

func ErrorPermisoDenegado(accion, etapa string, roles []Rol) *ErrorDominio {
	return NuevoError(CodigoPermisoDenegado,
		"Permiso denegado: la accion %q en la etapa %q requiere uno de estos roles: %v.",
		accion, etapa, roles)
}

func ErrorEstadoProyecto(mensaje string) *ErrorDominio {
	return NuevoError(CodigoEstadoProyecto, "%s", mensaje)
}
