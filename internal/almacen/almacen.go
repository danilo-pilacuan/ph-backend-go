// Package almacen: contrato del repositorio de dominio (espejo de
// app/almacen.py). El servicio accede al estado solo por esta interfaz:
// getters, *_agregar y *_reemplazar. Sin DSN cae a memoria con semilla.
package almacen

import (
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/semilla"
)

// Repositorio: contrato unico para memoria y Postgres. Las listas devueltas
// son copias: el servicio muta por *_reemplazar / *_agregar.
type Repositorio interface {
	Usuarios() []domain.Usuario
	Proyectos() []domain.Proyecto
	Etapas() []domain.EtapaTrabajo
	Entregables() []domain.Entregable
	Bitacora() []domain.BitacoraEntrada
	Relaciones() []domain.RelacionDocumento

	UsuariosReemplazar([]domain.Usuario)
	ProyectosReemplazar([]domain.Proyecto)
	EtapasReemplazar([]domain.EtapaTrabajo)
	EntregablesReemplazar([]domain.Entregable)
	BitacoraReemplazar([]domain.BitacoraEntrada)
	RelacionesReemplazar([]domain.RelacionDocumento)

	UsuariosAgregar(domain.Usuario)
	ProyectosAgregar(domain.Proyecto)
	EtapasAgregar(domain.EtapaTrabajo)
	EntregablesAgregar(domain.Entregable)
	BitacoraAgregar(domain.BitacoraEntrada)
	RelacionesAgregar(domain.RelacionDocumento)
}

// RepositorioMemoria: listas vivas en proceso (fallback dev y tests).
// ConSemilla=true repone las 5 colecciones desde semilla, igual que el Python.
type RepositorioMemoria struct {
	usuarios    []domain.Usuario
	proyectos   []domain.Proyecto
	etapas      []domain.EtapaTrabajo
	entregables []domain.Entregable
	bitacora    []domain.BitacoraEntrada
	relaciones  []domain.RelacionDocumento
}

func NuevoMemoria(conSemilla bool) *RepositorioMemoria {
	r := &RepositorioMemoria{}
	if conSemilla {
		r.CargarSemilla()
	}
	return r
}

func (r *RepositorioMemoria) CargarSemilla() {
	r.UsuariosReemplazar(semilla.UsuariosSemilla)
	r.ProyectosReemplazar(semilla.ProyectosSemilla)
	r.EtapasReemplazar(semilla.EtapasSemilla)
	r.EntregablesReemplazar(semilla.EntregablesSemilla)
	r.BitacoraReemplazar(semilla.BitacoraSemilla)
}

func copiar[T any](src []T) []T {
	out := make([]T, len(src))
	copy(out, src)
	return out
}

func (r *RepositorioMemoria) Usuarios() []domain.Usuario         { return copiar(r.usuarios) }
func (r *RepositorioMemoria) Proyectos() []domain.Proyecto       { return copiar(r.proyectos) }
func (r *RepositorioMemoria) Etapas() []domain.EtapaTrabajo      { return copiar(r.etapas) }
func (r *RepositorioMemoria) Entregables() []domain.Entregable   { return copiar(r.entregables) }
func (r *RepositorioMemoria) Bitacora() []domain.BitacoraEntrada { return copiar(r.bitacora) }
func (r *RepositorioMemoria) Relaciones() []domain.RelacionDocumento {
	return copiar(r.relaciones)
}

func (r *RepositorioMemoria) UsuariosReemplazar(l []domain.Usuario)         { r.usuarios = copiar(l) }
func (r *RepositorioMemoria) ProyectosReemplazar(l []domain.Proyecto)       { r.proyectos = copiar(l) }
func (r *RepositorioMemoria) EtapasReemplazar(l []domain.EtapaTrabajo)      { r.etapas = copiar(l) }
func (r *RepositorioMemoria) EntregablesReemplazar(l []domain.Entregable)   { r.entregables = copiar(l) }
func (r *RepositorioMemoria) BitacoraReemplazar(l []domain.BitacoraEntrada) { r.bitacora = copiar(l) }
func (r *RepositorioMemoria) RelacionesReemplazar(l []domain.RelacionDocumento) {
	r.relaciones = copiar(l)
}

func (r *RepositorioMemoria) UsuariosAgregar(e domain.Usuario) { r.usuarios = append(r.usuarios, e) }
func (r *RepositorioMemoria) ProyectosAgregar(e domain.Proyecto) {
	r.proyectos = append(r.proyectos, e)
}
func (r *RepositorioMemoria) EtapasAgregar(e domain.EtapaTrabajo) { r.etapas = append(r.etapas, e) }
func (r *RepositorioMemoria) EntregablesAgregar(e domain.Entregable) {
	r.entregables = append(r.entregables, e)
}
func (r *RepositorioMemoria) BitacoraAgregar(e domain.BitacoraEntrada) {
	r.bitacora = append(r.bitacora, e)
}
func (r *RepositorioMemoria) RelacionesAgregar(e domain.RelacionDocumento) {
	r.relaciones = append(r.relaciones, e)
}
