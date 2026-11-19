// Package domain: gate de etapa (espejo de gates.py, 3.4). No se avanza de
// etapa sin los entregables obligatorios aprobados: el gate es funcionalidad
// de primera clase, no un adorno.
package domain

// EntregablesObligatorios: documentos obligatorios por etapa; sin ellos el
// gate no abre. Mismos nombres exactos que la version Python/TS.
var EntregablesObligatorios = map[EtapaCicloVida][]string{
	EtapaPlanteamiento: {"Acta de planteamiento"},
	EtapaIniciativa:    {"Documento de iniciativa"},
	EtapaPoC:           {"Plan de prueba de concepto"},
	EtapaValidacionPoC: {"Informe de validacion de PoC"},
	EtapaPlaneacion:    {"Plan de proyecto"},
	EtapaContratos:     {"Contrato o acta de compromisos"},
	EtapaEjecucion:     {"Acta de avance de ejecucion"},
	EtapaOperacion:     {"Acta de operacion"},
	EtapaCierre:        {"Acta de cierre"},
}

// AprobadoresRequeridos: roles cuya aprobacion se exige para abrir el gate.
var AprobadoresRequeridos = []Rol{RolAdmin, RolSponsor}

// EvaluarGate: faltantes = obligatorios sin entregable; pendientes = con
// entregable cargado pero sin aprobacion.
func EvaluarGate(etapa EtapaCicloVida, entregables []Entregable) EstadoGate {
	obligatorios := EntregablesObligatorios[etapa]
	deEtapa := []Entregable{}
	for _, e := range entregables {
		if e.Etapa == etapa {
			deEtapa = append(deEtapa, e)
		}
	}
	faltantes := []string{}
	pendientes := []string{}
	for _, nombre := range obligatorios {
		cargado := false
		for _, e := range deEtapa {
			if e.Nombre == nombre {
				cargado = true
				if e.AprobadoPorID == nil {
					pendientes = append(pendientes, e.Nombre)
				}
			}
		}
		if !cargado {
			faltantes = append(faltantes, nombre)
		}
	}
	return EstadoGate{
		Etapa:                etapa,
		Entregables:          deEtapa,
		Faltantes:            faltantes,
		PendientesAprobacion: pendientes,
		Abierto:              len(faltantes) == 0 && len(pendientes) == 0,
	}
}

// ValidarAprobacion: el autor no aprueba su propio documento y un entregable
// ya aprobado no se reaprueba (3.4).
func ValidarAprobacion(entregable Entregable, aprobadorID string) error {
	if entregable.AprobadoPorID != nil {
		return ErrorValidacion("El entregable %q ya fue aprobado y no admite otra aprobacion.", entregable.Nombre)
	}
	if entregable.CreadoPorID == aprobadorID {
		return ErrorValidacion("El autor de un entregable no puede aprobar su propio documento.")
	}
	return nil
}
