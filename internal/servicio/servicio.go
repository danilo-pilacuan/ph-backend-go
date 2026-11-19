// Package servicio: casos de uso de dominio (espejo de app/servicio.py).
// El estado vive en almacen.Repositorio; el servicio nunca toca SQL.
package servicio

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

var contador int64

// NuevoID: "prefijo-YYYYMMDDHHMMSS-n", paridad con nuevo_id de servicio.py.
func NuevoID(prefijo string) string {
	contador++
	return fmt.Sprintf("%s-%s-%d", prefijo, time.Now().UTC().Format("20060102150405"), contador)
}

// ServicioProyectos: casos de uso sobre el repositorio inyectado.
type ServicioProyectos struct {
	repo       almacen.Repositorio
	credCambio CambioCredenciales
}

// CambioCredenciales: gancho de credenciales inyectado por main (renombrar y
// sincronizar huerfanas tras un re-sync total).
type CambioCredenciales interface {
	Renombrar(correoAnterior, correoNuevo string)
	SincronizarConUsuarios(correos map[string]bool)
}

func NuevoServicio(repo almacen.Repositorio) *ServicioProyectos {
	return &ServicioProyectos{repo: repo}
}

func (s *ServicioProyectos) AsignarCredenciales(c CambioCredenciales) { s.credCambio = c }

// ---------- Consultas ----------

func (s *ServicioProyectos) Usuarios() []domain.Usuario         { return s.repo.Usuarios() }
func (s *ServicioProyectos) Proyectos() []domain.Proyecto       { return s.repo.Proyectos() }
func (s *ServicioProyectos) Etapas() []domain.EtapaTrabajo      { return s.repo.Etapas() }
func (s *ServicioProyectos) Entregables() []domain.Entregable   { return s.repo.Entregables() }
func (s *ServicioProyectos) Bitacora() []domain.BitacoraEntrada { return s.repo.Bitacora() }

func (s *ServicioProyectos) UsuarioPorID(id string) *domain.Usuario {
	for i := range s.repo.Usuarios() {
		if s.repo.Usuarios()[i].ID == id {
			u := s.repo.Usuarios()[i]
			return &u
		}
	}
	return nil
}

func (s *ServicioProyectos) ProyectoPorID(id string) *domain.Proyecto {
	lista := s.repo.Proyectos()
	for i := range lista {
		if lista[i].ID == id {
			p := lista[i]
			return &p
		}
	}
	return nil
}

func (s *ServicioProyectos) EtapasDeProyecto(proyectoID string) []domain.EtapaTrabajo {
	var out []domain.EtapaTrabajo
	for _, e := range s.repo.Etapas() {
		if e.ProyectoID == proyectoID {
			out = append(out, e)
		}
	}
	return out
}

func (s *ServicioProyectos) EntregablesDeProyecto(proyectoID string) []domain.Entregable {
	var out []domain.Entregable
	for _, e := range s.repo.Entregables() {
		if e.ProyectoID == proyectoID {
			out = append(out, e)
		}
	}
	return out
}

func (s *ServicioProyectos) EtapaPorID(id string) *domain.EtapaTrabajo {
	lista := s.repo.Etapas()
	for i := range lista {
		if lista[i].ID == id {
			e := lista[i]
			return &e
		}
	}
	return nil
}

// ExigirEtapa: ENTIDAD_NO_ENCONTRADA si la etapa no existe (404).
func (s *ServicioProyectos) ExigirEtapa(id string) (domain.EtapaTrabajo, error) {
	e := s.EtapaPorID(id)
	if e == nil {
		return domain.EtapaTrabajo{}, domain.ErrorEntidadNoEncontrada("Etapa de trabajo", id)
	}
	return *e, nil
}

// GateActual: gate de etapa para la etapa vigente del proyecto (3.4).
func (s *ServicioProyectos) GateActual(proyectoID string) (domain.EstadoGate, error) {
	proyecto := s.ProyectoPorID(proyectoID)
	if proyecto == nil {
		return domain.EstadoGate{}, domain.ErrorEntidadNoEncontrada("Proyecto", proyectoID)
	}
	return domain.EvaluarGate(proyecto.Etapa, s.EntregablesDeProyecto(proyectoID)), nil
}

// Resumen: calculos agregados del proyecto (5.1-5.5).
func (s *ServicioProyectos) Resumen(proyectoID string) (domain.ResumenProyecto, error) {
	proyecto := s.ProyectoPorID(proyectoID)
	if proyecto == nil {
		return domain.ResumenProyecto{}, domain.ErrorEntidadNoEncontrada("Proyecto", proyectoID)
	}
	etapas := s.EtapasDeProyecto(proyectoID)
	avance, err := domain.AvancePonderado(etapas)
	if err != nil {
		return domain.ResumenProyecto{}, err
	}
	presupuesto, err := domain.PresupuestoTotal(etapas)
	if err != nil {
		return domain.ResumenProyecto{}, err
	}
	consumo, err := domain.ConsumoTotal(etapas)
	if err != nil {
		return domain.ResumenProyecto{}, err
	}
	excedido := false
	for _, e := range etapas {
		if domain.PresupuestoExcedido(e) {
			excedido = true
		}
	}
	return domain.ResumenProyecto{
		Proyecto:         *proyecto,
		Avance:           avance,
		PresupuestoTotal: presupuesto,
		ConsumoTotal:     consumo,
		Etapas:           etapas,
		Excedido:         excedido,
	}, nil
}

// Resumenes: resumen de todos los proyectos (5.1-5.5, espejo resumenes()).
func (s *ServicioProyectos) Resumenes() []domain.ResumenProyecto {
	out := []domain.ResumenProyecto{}
	for _, p := range s.repo.Proyectos() {
		if rs, err := s.Resumen(p.ID); err == nil {
			out = append(out, rs)
		}
	}
	return out
}

// BitacoraDeProyecto: entradas del proyecto (consulta de bitacora 3.5).
func (s *ServicioProyectos) BitacoraDeProyecto(proyectoID string) []domain.BitacoraEntrada {
	var out []domain.BitacoraEntrada
	for _, b := range s.repo.Bitacora() {
		if b.ProyectoID == proyectoID {
			out = append(out, b)
		}
	}
	return out
}

// ---------- Helpers privados ----------

func (s *ServicioProyectos) exigirProyecto(id string) (domain.Proyecto, error) {
	p := s.ProyectoPorID(id)
	if p == nil {
		return domain.Proyecto{}, domain.ErrorEntidadNoEncontrada("Proyecto", id)
	}
	return *p, nil
}

func (s *ServicioProyectos) exigirActorValido(actor domain.Usuario) error {
	if !actor.Activo {
		return domain.ErrorValidacion("Un usuario inactivo no puede ejecutar acciones (1.8).")
	}
	return nil
}

func (s *ServicioProyectos) exigirEstadoEditable(estado domain.EstadoProyecto, operacion string) error {
	if estado == domain.EstadoCerrado {
		return domain.ErrorEstadoProyecto("Un proyecto cerrado es de solo lectura.")
	}
	if estado == domain.EstadoCancelado && operacion != "registrar consumo" {
		return domain.ErrorEstadoProyecto("Un proyecto cancelado no admite esta operacion.")
	}
	return nil
}

func (s *ServicioProyectos) exigirResponsableActivo(responsableID string) error {
	u := s.UsuarioPorID(responsableID)
	if u == nil {
		return domain.ErrorEntidadNoEncontrada("Usuario", responsableID)
	}
	if !u.Activo {
		return domain.ErrorValidacion("El responsable debe ser un usuario activo.")
	}
	return nil
}

func (s *ServicioProyectos) existeNombre(nombre string, exceptoID string) bool {
	normalizado := strings.ToLower(nombre)
	for _, p := range s.repo.Proyectos() {
		if p.ID != exceptoID && strings.ToLower(p.Nombre) == normalizado {
			return true
		}
	}
	return false
}

func (s *ServicioProyectos) registrarBitacora(proyectoID, autorID string, tipo domain.TipoBitacora, origen, destino *domain.EtapaCicloVida, comentario *string) {
	entrada := domain.BitacoraEntrada{
		ID:         NuevoID("b"),
		ProyectoID: proyectoID,
		AutorID:    autorID,
		Fecha:      time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Tipo:       tipo,
		Comentario: comentario,
	}
	if origen != nil {
		v := *origen
		entrada.EtapaOrigen = &v
	}
	if destino != nil {
		v := *destino
		entrada.EtapaDestino = &v
	}
	s.repo.BitacoraAgregar(entrada)
}

// esMontoValido: espejo de las guardas de presupuesto de servicio.py.
func esMontoValido(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

var _ = os.Getenv // placeholder: se usa en fases posteriores (rate limit y CORS se configuran en main)
