// Package servicio: etapas de trabajo, entregables y usuarios (parte 3,
// espejo de app/servicio.py).
package servicio

import (
	"strings"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// DatosEtapa: entrada de crear/editar etapa (opcional en nil).
type DatosEtapa struct {
	Nombre              *string
	FechaInicio         *string
	FechaFin            *string
	ResponsableID       *string
	PresupuestoAsignado *float64
	Avance              *float64
	Dependencias        *[]string // nil = no tocar; &[] = reemplaza (6.5)
}

// CrearEtapa: valida fechas/montos/avance, responsable activo y dependencias
// candidatas antes de agregar; registro en bitacora.
func (s *ServicioProyectos) CrearEtapa(proyectoID string, datos DatosEtapa, actor domain.Usuario) (domain.EtapaTrabajo, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "crear etapas"); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionEditarEtapa, actor.Roles); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	etapa := domain.EtapaTrabajo{
		ID:         NuevoID("e"),
		ProyectoID: proyecto.ID,
	}
	if datos.Nombre != nil {
		etapa.Nombre = *datos.Nombre
	}
	if datos.FechaInicio != nil {
		etapa.FechaInicio = *datos.FechaInicio
	}
	if datos.FechaFin != nil {
		etapa.FechaFin = *datos.FechaFin
	}
	if datos.ResponsableID != nil {
		etapa.ResponsableID = *datos.ResponsableID
	}
	if datos.PresupuestoAsignado != nil {
		etapa.PresupuestoAsignado = *datos.PresupuestoAsignado
	}
	if datos.Avance != nil {
		etapa.Avance = *datos.Avance
	}
	if datos.Dependencias != nil {
		etapa.Dependencias = *datos.Dependencias
	}
	if err := s.validarDatosEtapa(etapa); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirResponsableActivo(etapa.ResponsableID); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.validarDependenciasCandidata(etapa); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	s.repo.EtapasAgregar(etapa)
	comentario := "Etapa de trabajo \"" + etapa.Nombre + "\" creada."
	s.registrarBitacora(proyecto.ID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return etapa, nil
}

// EditarEtapa: fusion con fusion candidata revalidada; dependencias nil no
// toca, [] explicito borra (6.5). Reexige prerequisitos al 100% si ya hay avance.
func (s *ServicioProyectos) EditarEtapa(etapaID string, datos DatosEtapa, actor domain.Usuario) (domain.EtapaTrabajo, error) {
	etapa, err := s.ExigirEtapa(etapaID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	proyecto, err := s.exigirProyecto(etapa.ProyectoID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "editar etapas"); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionEditarEtapa, actor.Roles); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if datos.Nombre != nil {
		etapa.Nombre = *datos.Nombre
	}
	if datos.FechaInicio != nil {
		etapa.FechaInicio = *datos.FechaInicio
	}
	if datos.FechaFin != nil {
		etapa.FechaFin = *datos.FechaFin
	}
	if datos.ResponsableID != nil {
		etapa.ResponsableID = *datos.ResponsableID
	}
	if datos.PresupuestoAsignado != nil {
		etapa.PresupuestoAsignado = *datos.PresupuestoAsignado
	}
	if datos.Avance != nil {
		etapa.Avance = *datos.Avance
	}
	if datos.Dependencias != nil {
		etapa.Dependencias = *datos.Dependencias
	}
	if err := s.validarDatosEtapa(etapa); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if datos.ResponsableID != nil {
		if err := s.exigirResponsableActivo(*datos.ResponsableID); err != nil {
			return domain.EtapaTrabajo{}, err
		}
	}
	if err := s.validarDependenciasCandidata(etapa); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirDependenciasCompletas(etapa, etapa.Avance, s.etapasDelProyectoLocal(proyecto.ID, etapa)); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	s.reemplazarEtapa(etapa)
	return etapa, nil
}

// EliminarEtapa: rechazada si otras etapas dependen de ella (6.5).
func (s *ServicioProyectos) EliminarEtapa(etapaID string, actor domain.Usuario) error {
	etapa, err := s.ExigirEtapa(etapaID)
	if err != nil {
		return err
	}
	proyecto, err := s.exigirProyecto(etapa.ProyectoID)
	if err != nil {
		return err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "eliminar etapas"); err != nil {
		return err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionEditarEtapa, actor.Roles); err != nil {
		return err
	}
	dependientes := domain.DependientesDe(etapaID, s.repo.Etapas())
	if len(dependientes) > 0 {
		nombres := ""
		for i, d := range dependientes {
			if i > 0 {
				nombres += ", "
			}
			nombres += d.Nombre
		}
		return domain.ErrorValidacion("No se puede eliminar la etapa %q: las etapas %s dependen de ella.", etapa.Nombre, nombres)
	}
	lista := s.repo.Etapas()
	out := lista[:0]
	for _, e := range lista {
		if e.ID != etapaID {
			out = append(out, e)
		}
	}
	s.repo.EtapasReemplazar(out)
	return nil
}

// RegistrarAvance: valida rango y dependencias al 100% antes de mutar.
func (s *ServicioProyectos) RegistrarAvance(etapaID string, avance float64, actor domain.Usuario) (domain.EtapaTrabajo, error) {
	etapa, err := s.ExigirEtapa(etapaID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	proyecto, err := s.exigirProyecto(etapa.ProyectoID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "registrar avance"); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionEditarEtapa, actor.Roles); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	candidata := etapa
	candidata.Avance = avance
	if err := s.validarDatosEtapa(candidata); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirDependenciasCompletas(candidata, avance, s.etapasDelProyectoLocal(proyecto.ID, candidata)); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	s.reemplazarEtapa(candidata)
	return candidata, nil
}

// RegistrarConsumo: unico caso de uso permitido sobre proyecto cancelado.
func (s *ServicioProyectos) RegistrarConsumo(etapaID string, consumo float64, actor domain.Usuario) (domain.EtapaTrabajo, error) {
	etapa, err := s.ExigirEtapa(etapaID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	proyecto, err := s.exigirProyecto(etapa.ProyectoID)
	if err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "registrar consumo"); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionRegistrarConsumo, actor.Roles); err != nil {
		return domain.EtapaTrabajo{}, err
	}
	if !esMontoValido(consumo) {
		return domain.EtapaTrabajo{}, domain.ErrorValidacion("El consumo no puede ser negativo.")
	}
	etapa.PresupuestoConsumido = consumo
	s.reemplazarEtapa(etapa)
	return etapa, nil
}

// CrearEntregable: nombre obligatorio (strip), version >= 1, etapa del gate.
func (s *ServicioProyectos) CrearEntregable(proyectoID string, etapaCiclo domain.EtapaCicloVida, nombre string, version int, actor domain.Usuario) (domain.Entregable, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.Entregable{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Entregable{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "cargar entregables"); err != nil {
		return domain.Entregable{}, err
	}
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return domain.Entregable{}, domain.ErrorValidacion("El nombre del entregable es obligatorio.")
	}
	if version < 1 {
		return domain.Entregable{}, domain.ErrorValidacion("La version del entregable debe ser al menos 1.")
	}
	entregable := domain.Entregable{
		ID:          NuevoID("ent"),
		ProyectoID:  proyecto.ID,
		Etapa:       etapaCiclo,
		Nombre:      nombre,
		Version:     version,
		CreadoPorID: actor.ID,
	}
	s.repo.EntregablesAgregar(entregable)
	comentario := "Entregable \"" + entregable.Nombre + "\" cargado (v" + itoa(version) + ")."
	s.registrarBitacora(proyecto.ID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return entregable, nil
}

// AprobarEntregable: gate 3.4: actor activo, permiso por matriz, autor no
// se autoaprueba, no se reaprueba; fecha ISO del servidor.
func (s *ServicioProyectos) AprobarEntregable(entregableID string, actor domain.Usuario) (domain.Entregable, error) {
	lista := s.repo.Entregables()
	var entregable *domain.Entregable
	for i := range lista {
		if lista[i].ID == entregableID {
			entregable = &lista[i]
			break
		}
	}
	if entregable == nil {
		return domain.Entregable{}, domain.ErrorEntidadNoEncontrada("Entregable", entregableID)
	}
	proyecto, err := s.exigirProyecto(entregable.ProyectoID)
	if err != nil {
		return domain.Entregable{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Entregable{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "aprobar entregables"); err != nil {
		return domain.Entregable{}, err
	}
	if err := domain.ExigirPermiso(entregable.Etapa, domain.AccionAprobarEntregable, actor.Roles); err != nil {
		return domain.Entregable{}, err
	}
	if err := domain.ValidarAprobacion(*entregable, actor.ID); err != nil {
		return domain.Entregable{}, err
	}
	ahora := time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")
	entregable.AprobadoPorID = &actor.ID
	entregable.AprobadoEn = &ahora
	s.repo.EntregablesReemplazar(lista)
	comentario := "Entregable \"" + entregable.Nombre + "\" aprobado (v" + itoa(entregable.Version) + ")."
	s.registrarBitacora(proyecto.ID, actor.ID, domain.TipoAprobacion, nil, nil, &comentario)
	return *entregable, nil
}

func (s *ServicioProyectos) validarDatosEtapa(e domain.EtapaTrabajo) error {
	return domain.ValidarEtapa(e.FechaInicio, e.FechaFin, e.Avance, e.PresupuestoAsignado, e.PresupuestoConsumido)
}

func (s *ServicioProyectos) validarDependenciasCandidata(e domain.EtapaTrabajo) error {
	delProyecto := s.etapasDelProyectoLocal(e.ProyectoID, e)
	return domain.ValidarDependencias(e.ID, e.ProyectoID, delProyecto)
}

// etapasDelProyectoLocal: etapas del proyecto con la candidata sustituida.
func (s *ServicioProyectos) etapasDelProyectoLocal(proyectoID string, candidata domain.EtapaTrabajo) []domain.EtapaTrabajo {
	var out []domain.EtapaTrabajo
	for _, e := range s.repo.Etapas() {
		if e.ProyectoID != proyectoID {
			continue
		}
		if e.ID == candidata.ID {
			out = append(out, candidata)
		} else {
			out = append(out, e)
		}
	}
	if candidata.ID != "" && !contieneID(out, candidata.ID) {
		out = append(out, candidata)
	}
	return out
}

func contieneID(etapas []domain.EtapaTrabajo, id string) bool {
	for _, e := range etapas {
		if e.ID == id {
			return true
		}
	}
	return false
}

func (s *ServicioProyectos) reemplazarEtapa(etapa domain.EtapaTrabajo) {
	lista := s.repo.Etapas()
	for i := range lista {
		if lista[i].ID == etapa.ID {
			lista[i] = etapa
			break
		}
	}
	s.repo.EtapasReemplazar(lista)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digitos := ""
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		digitos = string(rune('0'+v%10)) + digitos
		v /= 10
	}
	if neg {
		return "-" + digitos
	}
	return digitos
}
