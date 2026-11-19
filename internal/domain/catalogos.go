// Package domain: catalogos de dominio (espejo de shared-lib/domain/catalogos.ts).
// Los valores string son parte del contrato JSON camelCase con @projecthub/shared
// y la API FastAPI actual: no se renombran.
package domain

type EstadoProyecto string

const (
	EstadoActivo    EstadoProyecto = "Activo"
	EstadoCancelado EstadoProyecto = "Cancelado"
	EstadoCerrado   EstadoProyecto = "Cerrado"
)

// Etapas del ciclo de vida (3.1): "Iniciativa" es una etapa, no una entidad.
type EtapaCicloVida string

const (
	EtapaPlanteamiento EtapaCicloVida = "Planteamiento"
	EtapaIniciativa    EtapaCicloVida = "Iniciativa"
	EtapaPoC           EtapaCicloVida = "PoC"
	EtapaValidacionPoC EtapaCicloVida = "ValidacionPoC"
	EtapaPlaneacion    EtapaCicloVida = "Planeacion"
	EtapaContratos     EtapaCicloVida = "Contratos"
	EtapaEjecucion     EtapaCicloVida = "Ejecucion"
	EtapaOperacion     EtapaCicloVida = "Operacion"
	EtapaCierre        EtapaCicloVida = "Cierre"
)

var EtapasCicloVida = []EtapaCicloVida{
	EtapaPlanteamiento,
	EtapaIniciativa,
	EtapaPoC,
	EtapaValidacionPoC,
	EtapaPlaneacion,
	EtapaContratos,
	EtapaEjecucion,
	EtapaOperacion,
	EtapaCierre,
}

type Rol string

const (
	RolAdmin          Rol = "Admin"
	RolSponsor        Rol = "Sponsor"
	RolProjectManager Rol = "ProjectManager"
	RolAnalista       Rol = "Analista"
	RolStakeholder    Rol = "Stakeholder"
)

type Accion string

const (
	AccionVerProyecto       Accion = "verProyecto"
	AccionEditarProyecto    Accion = "editarProyecto"
	AccionAvanzarEtapa      Accion = "avanzarEtapa"
	AccionRetrocederEtapa   Accion = "retrocederEtapa"
	AccionAprobarEntregable Accion = "aprobarEntregable"
	AccionEditarEtapa       Accion = "editarEtapaTrabajo"
	AccionEditarPresupuesto Accion = "editarPresupuesto"
	AccionRegistrarConsumo  Accion = "registrarConsumo"
	AccionCancelarProyecto  Accion = "cancelarProyecto"
	AccionCerrarProyecto    Accion = "cerrarProyecto"
)

var IndiceEtapa = func() map[EtapaCicloVida]int {
	m := make(map[EtapaCicloVida]int, len(EtapasCicloVida))
	for i, e := range EtapasCicloVida {
		m[e] = i
	}
	return m
}()
