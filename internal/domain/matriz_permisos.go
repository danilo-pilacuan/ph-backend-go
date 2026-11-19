// Package domain: matriz de responsabilidades (espejo de matriz_permisos.py,
// 1.5). Fuente unica para exigir_permiso y para los tests de permisos.
package domain

type rolesSet map[Rol]bool

func roles(rols ...Rol) []Rol { return rols }

var TodosLosRoles = []Rol{RolAdmin, RolSponsor, RolProjectManager, RolAnalista, RolStakeholder}

var (
	SoloGobernanza = []Rol{RolAdmin, RolSponsor}
	GobernanzaYPM  = []Rol{RolAdmin, RolSponsor, RolProjectManager}
	Ninguno        = []Rol{}
)

// Matriz: etapa -> accion -> roles habilitados. Mismos valores que la version
// Python; cualquier cambio se propaga a tests de ambos lados.
var Matriz = map[EtapaCicloVida]map[Accion][]Rol{
	EtapaPlanteamiento: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      GobernanzaYPM,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: GobernanzaYPM,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaIniciativa: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      GobernanzaYPM,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: GobernanzaYPM,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaPoC: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   SoloGobernanza,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: GobernanzaYPM,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaValidacionPoC: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   SoloGobernanza,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: GobernanzaYPM,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaPlaneacion: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: GobernanzaYPM,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaContratos: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    SoloGobernanza,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       SoloGobernanza,
		AccionEditarPresupuesto: SoloGobernanza,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaEjecucion: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: SoloGobernanza,
		AccionRegistrarConsumo:  GobernanzaYPM,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaOperacion: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    GobernanzaYPM,
		AccionAvanzarEtapa:      SoloGobernanza,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       GobernanzaYPM,
		AccionEditarPresupuesto: SoloGobernanza,
		AccionRegistrarConsumo:  GobernanzaYPM,
		AccionCancelarProyecto:  SoloGobernanza,
		AccionCerrarProyecto:    Ninguno,
	},
	EtapaCierre: {
		AccionVerProyecto:       TodosLosRoles,
		AccionEditarProyecto:    Ninguno,
		AccionAvanzarEtapa:      Ninguno,
		AccionRetrocederEtapa:   Ninguno,
		AccionAprobarEntregable: SoloGobernanza,
		AccionEditarEtapa:       Ninguno,
		AccionEditarPresupuesto: Ninguno,
		AccionRegistrarConsumo:  Ninguno,
		AccionCancelarProyecto:  Ninguno,
		AccionCerrarProyecto:    SoloGobernanza,
	},
}

// TienePermiso: true si alguno de los roles del actor esta habilitado.
func TienePermiso(etapa EtapaCicloVida, accion Accion, roles []Rol) bool {
	permitidos := Matriz[etapa][accion]
	for _, r := range roles {
		for _, p := range permitidos {
			if r == p {
				return true
			}
		}
	}
	return false
}

// ExigirPermiso: error PERMISO_DENEGADO (409) si ningun rol aplica.
func ExigirPermiso(etapa EtapaCicloVida, accion Accion, roles []Rol) error {
	if TienePermiso(etapa, accion, roles) {
		return nil
	}
	return ErrorPermisoDenegado(string(accion), string(etapa), Matriz[etapa][accion])
}
