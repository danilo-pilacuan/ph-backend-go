// Package domain: modelos de dominio (espejo de domain/modelos.py y
// shared-lib/domain/modelos.ts). Los tags JSON fijan el contrato camelCase
// identico: fechas ISO como string, sin time.Time para no alterar el formato.
package domain

// Usuario del sistema; Roles usa los valores exactos del catalogo.
type Usuario struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Correo string `json:"correo"`
	Roles  []Rol  `json:"roles"`
	Activo bool   `json:"activo"`
}

// DependenciasVacias: contracto Python serializa dependencias: [] (no null).
func DependenciasVacias() []string { return []string{} }

// EtapaTrabajo: unidad de planificacion (lo que se dibuja en el Gantt).
type EtapaTrabajo struct {
	ID                   string   `json:"id"`
	ProyectoID           string   `json:"proyectoId"`
	Nombre               string   `json:"nombre"`
	FechaInicio          string   `json:"fechaInicio"` // ISO yyyy-mm-dd
	FechaFin             string   `json:"fechaFin"`    // ISO yyyy-mm-dd
	ResponsableID        string   `json:"responsableId"`
	PresupuestoAsignado  float64  `json:"presupuestoAsignado"`
	PresupuestoConsumido float64  `json:"presupuestoConsumido"`
	Avance               float64  `json:"avance"` // 0-100; el avance del proyecto se calcula, nunca se edita
	Dependencias         []string `json:"dependencias"`
}

// Normalizar: aplica invariantes de contrato tras cargar desde JSON externo
// (dependencias [] en lugar de null, igual que el valor por defecto de Python).
func (e *EtapaTrabajo) Normalizar() {
	if e.Dependencias == nil {
		e.Dependencias = []string{}
	}
}

// Proyecto: agregado central del sistema documental.
type Proyecto struct {
	ID                  string         `json:"id"`
	Nombre              string         `json:"nombre"`
	Descripcion         string         `json:"descripcion"`
	PatrocinadorID      string         `json:"patrocinadorId"`
	PresupuestoEstimado float64        `json:"presupuestoEstimado"`
	FechaObjetivo       string         `json:"fechaObjetivo"` // ISO yyyy-mm-dd
	Etapa               EtapaCicloVida `json:"etapa"`
	Estado              EstadoProyecto `json:"estado"`
}

// Entregable: documento obligatorio de una etapa, con version y aprobacion.
type Entregable struct {
	ID            string         `json:"id"`
	ProyectoID    string         `json:"proyectoId"`
	Etapa         EtapaCicloVida `json:"etapa"`
	Nombre        string         `json:"nombre"`
	Version       int            `json:"version"`
	AprobadoPorID *string        `json:"aprobadoPorId"` // nil si aun no se aprueba
	AprobadoEn    *string        `json:"aprobadoEn"`    // ISO datetime
	CreadoPorID   string         `json:"creadoPorId"`
}

// TipoBitacora: transicion, aprobacion o cambio (invariante 3.5).
type TipoBitacora string

const (
	TipoTransicion TipoBitacora = "transicion"
	TipoAprobacion TipoBitacora = "aprobacion"
	TipoCambio     TipoBitacora = "cambio"
)

// BitacoraEntrada: registro inmutable y con autor de acciones de gobierno.
type BitacoraEntrada struct {
	ID           string          `json:"id"`
	ProyectoID   string          `json:"proyectoId"`
	AutorID      string          `json:"autorId"`
	Fecha        string          `json:"fecha"` // ISO datetime
	Tipo         TipoBitacora    `json:"tipo"`
	EtapaOrigen  *EtapaCicloVida `json:"etapaOrigen"`
	EtapaDestino *EtapaCicloVida `json:"etapaDestino"`
	Comentario   *string         `json:"comentario"`
}

// ResumenProyecto: calculos agregados (5.1-5.5); nada se edita a mano.
type ResumenProyecto struct {
	Proyecto         Proyecto       `json:"proyecto"`
	Avance           float64        `json:"avance"`
	PresupuestoTotal float64        `json:"presupuestoTotal"`
	ConsumoTotal     float64        `json:"consumoTotal"`
	Etapas           []EtapaTrabajo `json:"etapas"`
	Excedido         bool           `json:"excedido"`
}

// EstadoGate: resultado de evaluar el gate de etapa (3.4).
type EstadoGate struct {
	Etapa                EtapaCicloVida `json:"etapa"`
	Abierto              bool           `json:"abierto"`
	Entregables          []Entregable   `json:"entregables"`
	Faltantes            []string       `json:"faltantes"`
	PendientesAprobacion []string       `json:"pendientesAprobacion"`
}
