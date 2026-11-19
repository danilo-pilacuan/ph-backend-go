// Package domain: modelos de dominio (espejo de domain/modelos.py y
// shared-lib/domain/modelos.ts). Los tags JSON fijan el contrato camelCase
// identico: fechas ISO como string, sin time.Time para no alterar el formato.
package domain

import "strings"

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

// TipoRelacion: semantica del enlace entre documentos (16.1). Los valores son
// parte del contrato JSON; el grafo de busqueda los usa como etiqueta de arista.
type TipoRelacion string

const (
	RelacionDerivaDe    TipoRelacion = "derivaDe"    // el destino deriva del origen
	RelacionModifica    TipoRelacion = "modifica"    // adenda/modificacion: destino modifica origen
	RelacionAnula       TipoRelacion = "anula"       // destino anula el origen
	RelacionRelacionado TipoRelacion = "relacionado" // enlace debil de referencia
	RelacionAclara      TipoRelacion = "aclara"      // aclaracion/acta que explica el origen
)

var TiposRelacion = []TipoRelacion{RelacionDerivaDe, RelacionModifica, RelacionAnula, RelacionRelacionado, RelacionAclara}

var EtiquetaTipoRelacion = map[TipoRelacion]string{
	RelacionDerivaDe:    "Deriva de",
	RelacionModifica:    "Modifica (adenda)",
	RelacionAnula:       "Anula",
	RelacionRelacionado: "Relacionado con",
	RelacionAclara:      "Aclara",
}

// EsTipoRelacionValido: el tipo llega del cliente; se valida en el canal.
func EsTipoRelacionValido(t TipoRelacion) bool {
	for _, v := range TiposRelacion {
		if v == t {
			return true
		}
	}
	return false
}

// RelacionDocumento: enlace tipado entre dos documentos del mismo proyecto
// (16.1, ideas del dueno: relaciones tipo obsidian para un grafo de busqueda
// de acuerdos, adendas y relaciones). Direccion: origen -> destino. El
// `rel` es 1:1 con los entregables (mismo proyecto): asi el grafo se arma
// sobre el catalogo documental existente sin duplicar titulos.
type RelacionDocumento struct {
	ID          string       `json:"id"`
	ProyectoID  string       `json:"proyectoId"`
	OrigenID    string       `json:"origenId"`  // entregable origen (ent-*)
	DestinoID   string       `json:"destinoId"` // entregable destino (ent-*)
	Tipo        TipoRelacion `json:"tipo"`
	CreadoPorID string       `json:"creadoPorId"`
	CreadoEn    string       `json:"creadoEn"` // ISO datetime del servidor
	Nota        *string      `json:"nota"`     // opcional: por que existe el enlace
}

// ValidarRelacion: invariante de dominio del enlace (16.1).
//   - tipo del catalogo; nota opcional acotada (<= 500 chars tras strip);
//   - origen y destino distintos (no hay autoenlace);
//   - el par (origen, destino, tipo) es unico (sin duplicar aristas);
//   - direccion consistente: si ya existe la inversa con tipo "modifica"/
//     "anula"/"derivaDe", se rechaza (el par solo puede enlazarse una vez).
func ValidarRelacion(r RelacionDocumento, existentes []RelacionDocumento, entregablesDelProyecto []Entregable) error {
	if !EsTipoRelacionValido(r.Tipo) {
		return ErrorValidacion("El tipo de relacion %q no es valido.", string(r.Tipo))
	}
	if r.OrigenID == r.DestinoID {
		return ErrorValidacion("Un documento no puede relacionarse consigo mismo.")
	}
	if len([]rune(strings.TrimSpace(derefString(r.Nota)))) > 500 {
		return ErrorValidacion("La nota de la relacion no puede exceder 500 caracteres.")
	}
	existeDoc := func(id string) bool {
		for _, e := range entregablesDelProyecto {
			if e.ID == id {
				return true
			}
		}
		return false
	}
	if !existeDoc(r.OrigenID) {
		return ErrorEntidadNoEncontrada("Entregable (origen de la relacion)", r.OrigenID)
	}
	if !existeDoc(r.DestinoID) {
		return ErrorEntidadNoEncontrada("Entregable (destino de la relacion)", r.DestinoID)
	}
	for _, previa := range existentes {
		if previa.ID == r.ID {
			continue
		}
		mismoPar := (previa.OrigenID == r.OrigenID && previa.DestinoID == r.DestinoID) ||
			(previa.OrigenID == r.DestinoID && previa.DestinoID == r.OrigenID)
		if mismoPar {
			return ErrorDuplicado("relacion", "par de documentos", r.OrigenID+" <-> "+r.DestinoID)
		}
	}
	return nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
