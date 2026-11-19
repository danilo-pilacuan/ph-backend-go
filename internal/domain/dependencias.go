// Package domain: reglas de dependencias entre etapas de trabajo (espejo de
// dependencias.py, 6.5). Una etapa puede declarar prerequisitos: etapas del
// mismo proyecto que deben estar al 100% antes de registrar avance mayor que 0.
package domain

// DependientesDe: etapas que declaran a etapaID como prerequisito.
func DependientesDe(etapaID string, etapas []EtapaTrabajo) []EtapaTrabajo {
	var out []EtapaTrabajo
	for _, e := range etapas {
		for _, d := range e.Dependencias {
			if d == etapaID {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// ValidarDependencias: sin self, sin duplicados, prerequisitos existentes del
// mismo proyecto y sin ciclos.
func ValidarDependencias(etapaID, proyectoID string, etapasDelProyecto []EtapaTrabajo) error {
	var etapa *EtapaTrabajo
	for i := range etapasDelProyecto {
		if etapasDelProyecto[i].ID == etapaID {
			etapa = &etapasDelProyecto[i]
			break
		}
	}
	if etapa == nil {
		return ErrorValidacion("La etapa %q no pertenece al proyecto indicado.", etapaID)
	}
	vistas := map[string]bool{}
	for _, prereq := range etapa.Dependencias {
		if prereq == etapaID {
			return ErrorValidacion("Una etapa no puede depender de si misma.")
		}
		if vistas[prereq] {
			return ErrorValidacion("La etapa declara el prerequisito %q mas de una vez.", prereq)
		}
		vistas[prereq] = true
		var existe *EtapaTrabajo
		for i := range etapasDelProyecto {
			if etapasDelProyecto[i].ID == prereq {
				existe = &etapasDelProyecto[i]
				break
			}
		}
		if existe == nil {
			return ErrorValidacion("El prerequisito %q no existe o no pertenece al proyecto.", prereq)
		}
		if existe.ProyectoID != proyectoID {
			return ErrorValidacion("Las dependencias solo se permiten entre etapas del mismo proyecto.")
		}
	}
	if TieneCiclo(etapaID, etapasDelProyecto) {
		return ErrorValidacion("Las dependencias forman un ciclo entre etapas.")
	}
	return nil
}

// TieneCiclo: grafo dirigido prerequisito -> dependiente; DFS con pila.
func TieneCiclo(etapaID string, etapas []EtapaTrabajo) bool {
	aristas := map[string][]string{}
	for _, e := range etapas {
		for _, prereq := range e.Dependencias {
			aristas[prereq] = append(aristas[prereq], e.ID)
		}
	}
	visitados := map[string]bool{}
	pila := []string{etapaID}
	enPila := map[string]bool{etapaID: true}
	for len(pila) > 0 {
		actual := pila[len(pila)-1]
		if visitados[actual] {
			pila = pila[:len(pila)-1]
			enPila[actual] = false
			continue
		}
		visitados[actual] = true
		for _, sig := range aristas[actual] {
			if sig == etapaID || enPila[sig] {
				return true
			}
			if !visitados[sig] {
				pila = append(pila, sig)
				enPila[sig] = true
			}
		}
	}
	return false
}

// ExigirDependenciasCompletas: si etapa registra avance > 0, cada prerequisito
// debe estar al 100% (protege contra relajar dependencias de una etapa avanzada).
func ExigirDependenciasCompletas(etapa EtapaTrabajo, avanceNuevo float64, etapas []EtapaTrabajo) error {
	if avanceNuevo <= 0 {
		return nil
	}
	for _, prereqID := range etapa.Dependencias {
		var prereq *EtapaTrabajo
		for i := range etapas {
			if etapas[i].ID == prereqID {
				prereq = &etapas[i]
				break
			}
		}
		if prereq == nil {
			continue // ya validado al declarar; ignora ids huerfanos defensivamente.
		}
		if prereq.Avance < 100 {
			return ErrorValidacion(
				"No se puede registrar avance en %q: su prerequisito %q tiene %.0f%% y requiere 100%%.",
				etapa.Nombre, prereq.Nombre, prereq.Avance)
		}
	}
	return nil
}

// DependenciasGantt: aristas {prerequisitoId, dependienteId} para el Gantt.
func DependenciasGantt(etapas []EtapaTrabajo) []map[string]string {
	var aristas []map[string]string
	ids := map[string]bool{}
	for _, e := range etapas {
		ids[e.ID] = true
	}
	for _, e := range etapas {
		for _, prereq := range e.Dependencias {
			if ids[prereq] {
				aristas = append(aristas, map[string]string{"prerequisitoId": prereq, "dependienteId": e.ID})
			}
		}
	}
	return aristas
}
