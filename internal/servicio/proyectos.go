// Package servicio: casos de uso de proyectos, etapas y entregables (parte 2,
// espejo de app/servicio.py). Reglas identicas: permisos por matriz, gate
// cerrado bloquea avance, bitacora inmutable con autor.
package servicio

import (
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// DatosProyecto: entrada de crear/editar (campos opcionales en nil).
type DatosProyecto struct {
	Nombre              *string
	Descripcion         *string
	PatrocinadorID      *string
	PresupuestoEstimado *float64
	FechaObjetivo       *string
}

// CrearProyecto: etapa inicial Planteamiento, estado Activo, unicidad de
// nombre case-insensitive y presupuesto no negativo.
func (s *ServicioProyectos) CrearProyecto(datos DatosProyecto, actor domain.Usuario) (domain.Proyecto, error) {
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Proyecto{}, err
	}
	if err := domain.ExigirPermiso(domain.EtapaPlanteamiento, domain.AccionEditarProyecto, actor.Roles); err != nil {
		return domain.Proyecto{}, err
	}
	nombre := ""
	if datos.Nombre != nil {
		nombre = *datos.Nombre
	}
	if nombre == "" {
		return domain.Proyecto{}, domain.ErrorValidacion("El nombre del proyecto es obligatorio.")
	}
	if s.existeNombre(nombre, "") {
		return domain.Proyecto{}, domain.ErrorDuplicado("proyecto", "nombre", nombre)
	}
	presupuestoEstimado := 0.0
	if datos.PresupuestoEstimado != nil {
		presupuestoEstimado = *datos.PresupuestoEstimado
		if !esMontoValido(presupuestoEstimado) {
			return domain.Proyecto{}, domain.ErrorValidacion("El presupuesto estimado no puede ser negativo.")
		}
	}
	patrocinador := actor.ID
	if datos.PatrocinadorID != nil && *datos.PatrocinadorID != "" {
		patrocinador = *datos.PatrocinadorID
	}
	descripcion := ""
	if datos.Descripcion != nil {
		descripcion = *datos.Descripcion
	}
	fechaObjetivo := ""
	if datos.FechaObjetivo != nil {
		fechaObjetivo = *datos.FechaObjetivo
	}
	proyecto := domain.Proyecto{
		ID:                  NuevoID("p"),
		Nombre:              nombre,
		Descripcion:         descripcion,
		PatrocinadorID:      patrocinador,
		PresupuestoEstimado: presupuestoEstimado,
		FechaObjetivo:       fechaObjetivo,
		Etapa:               domain.EtapaPlanteamiento,
		Estado:              domain.EstadoActivo,
	}
	s.repo.ProyectosAgregar(proyecto)
	comentario := "Proyecto creado en etapa Planteamiento."
	s.registrarBitacora(proyecto.ID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return proyecto, nil
}

// EditarProyecto: nombre/descripcion/fechaObjetivo; proyecto no cerrado ni
// cancelado (registrar consumo es la unica operacion permitida en cancelado).
func (s *ServicioProyectos) EditarProyecto(proyectoID string, datos DatosProyecto, actor domain.Usuario) (domain.Proyecto, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.Proyecto{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Proyecto{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "editar el proyecto"); err != nil {
		return domain.Proyecto{}, err
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionEditarProyecto, actor.Roles); err != nil {
		return domain.Proyecto{}, err
	}
	if datos.Nombre != nil {
		nombre := *datos.Nombre
		if nombre == "" {
			return domain.Proyecto{}, domain.ErrorValidacion("El nombre del proyecto es obligatorio.")
		}
		if s.existeNombre(nombre, proyectoID) {
			return domain.Proyecto{}, domain.ErrorDuplicado("proyecto", "nombre", nombre)
		}
		proyecto.Nombre = nombre
	}
	if datos.Descripcion != nil {
		proyecto.Descripcion = *datos.Descripcion
	}
	if datos.FechaObjetivo != nil {
		proyecto.FechaObjetivo = *datos.FechaObjetivo
	}
	s.reemplazarProyecto(proyecto)
	comentario := "Proyecto editado."
	s.registrarBitacora(proyectoID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return proyecto, nil
}

// CancelarProyecto: no permitido en Cierre (usar cerrar) ni ya cerrado.
func (s *ServicioProyectos) CancelarProyecto(proyectoID string, actor domain.Usuario, comentario *string) (domain.Proyecto, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.Proyecto{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Proyecto{}, err
	}
	if proyecto.Estado == domain.EstadoCerrado {
		return domain.Proyecto{}, domain.ErrorEstadoProyecto("Un proyecto cerrado es de solo lectura.")
	}
	if proyecto.Etapa == domain.EtapaCierre {
		return domain.Proyecto{}, domain.ErrorEstadoProyecto("No se puede cancelar un proyecto en etapa Cierre; use cerrar proyecto.")
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, domain.AccionCancelarProyecto, actor.Roles); err != nil {
		return domain.Proyecto{}, err
	}
	proyecto.Estado = domain.EstadoCancelado
	s.reemplazarProyecto(proyecto)
	if comentario == nil {
		v := "Proyecto cancelado."
		comentario = &v
	}
	s.registrarBitacora(proyectoID, actor.ID, domain.TipoCambio, nil, nil, comentario)
	return proyecto, nil
}

// CerrarProyecto: solo desde etapa Cierre, con permiso cerrarProyecto.
func (s *ServicioProyectos) CerrarProyecto(proyectoID string, actor domain.Usuario, comentario *string) (domain.Proyecto, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.Proyecto{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Proyecto{}, err
	}
	if proyecto.Etapa != domain.EtapaCierre {
		return domain.Proyecto{}, domain.ErrorEstadoProyecto("Solo se puede cerrar un proyecto en etapa Cierre.")
	}
	if proyecto.Estado == domain.EstadoCerrado {
		return domain.Proyecto{}, domain.ErrorEstadoProyecto("El proyecto ya esta cerrado.")
	}
	if err := domain.ExigirPermiso(domain.EtapaCierre, domain.AccionCerrarProyecto, actor.Roles); err != nil {
		return domain.Proyecto{}, err
	}
	proyecto.Estado = domain.EstadoCerrado
	s.reemplazarProyecto(proyecto)
	if comentario == nil {
		v := "Proyecto cerrado."
		comentario = &v
	}
	s.registrarBitacora(proyectoID, actor.ID, domain.TipoCambio, nil, nil, comentario)
	return proyecto, nil
}

// Transicionar: valida transicion del grafo, permiso (avanzar/retroceder) y
// gate abierto; la bitacora registra origen/destino (tipo transicion).
func (s *ServicioProyectos) Transicionar(proyectoID string, destino domain.EtapaCicloVida, actor domain.Usuario, comentario *string) (domain.Proyecto, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.Proyecto{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.Proyecto{}, err
	}
	if proyecto.Estado != domain.EstadoActivo {
		return domain.Proyecto{}, domain.ErrorEstadoProyecto("Un proyecto " + string(proyecto.Estado) + " no admite transiciones.")
	}
	if err := domain.ValidarTransicion(proyecto.Etapa, destino); err != nil {
		return domain.Proyecto{}, err
	}
	accion := domain.AccionAvanzarEtapa
	if domain.IndiceEtapa[destino] < domain.IndiceEtapa[proyecto.Etapa] {
		accion = domain.AccionRetrocederEtapa
	}
	if err := domain.ExigirPermiso(proyecto.Etapa, accion, actor.Roles); err != nil {
		return domain.Proyecto{}, err
	}
	gate := domain.EvaluarGate(proyecto.Etapa, s.EntregablesDeProyecto(proyectoID))
	if !gate.Abierto {
		return domain.Proyecto{}, domain.ErrorValidacion(
			"El gate de %q esta cerrado: faltan %d entregable(s) y hay %d sin aprobar.",
			string(proyecto.Etapa), len(gate.Faltantes), len(gate.PendientesAprobacion))
	}
	origen := proyecto.Etapa
	proyecto.Etapa = destino
	s.reemplazarProyecto(proyecto)
	s.registrarBitacora(proyectoID, actor.ID, domain.TipoTransicion, &origen, &destino, comentario)
	return proyecto, nil
}

func (s *ServicioProyectos) reemplazarProyecto(proyecto domain.Proyecto) {
	lista := s.repo.Proyectos()
	for i := range lista {
		if lista[i].ID == proyecto.ID {
			lista[i] = proyecto
			break
		}
	}
	s.repo.ProyectosReemplazar(lista)
}

var _ = time.Now // conserva import para formatos de fecha en fases siguientes
