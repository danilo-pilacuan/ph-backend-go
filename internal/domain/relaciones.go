// Package domain: grafo de relaciones entre documentos (16.1, espejo de la
// idea del dueno: relaciones tipo obsidian para busquedas de acuerdos,
// adendas y relaciones). Puro y sin framework; la UI pide el grafo y lo
// dibuja con SVG propio.
package domain

// NodoGrafo: documento con grado de conexiones (para resaltar hubs).
type NodoGrafo struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Etapa    string `json:"etapa"`
	Aprobado bool   `json:"aprobado"`
	Grado    int    `json:"grado"`
}

// AristaGrafo: enlace tipado dirigido (visualizacion puede ignorar direccion).
type AristaGrafo struct {
	ID      string       `json:"id"`
	Origen  string       `json:"origenId"`
	Destino string       `json:"destinoId"`
	Tipo    TipoRelacion `json:"tipo"`
}

// GrafoRelaciones: subgrafo inducido por los entregables del proyecto.
type GrafoRelaciones struct {
	Nodos   []NodoGrafo   `json:"nodos"`
	Aristas []AristaGrafo `json:"aristas"`
}

// GrafoDeRelaciones: arma el grafo del proyecto a partir de sus entregables
// y relaciones. Ignora relaciones huerfanas (documento eliminado) para que
// el grafo degrada con elegancia; el grado de un nodo cuenta aristas
// entrantes y salientes (conexiones totales, como el graph view de obsidian).
func GrafoDeRelaciones(entregables []Entregable, relaciones []RelacionDocumento) GrafoRelaciones {
	ids := map[string]bool{}
	nombres := map[string]string{}
	etapas := map[string]EntregaEtapa{}
	for _, e := range entregables {
		ids[e.ID] = true
		nombres[e.ID] = e.Nombre
		etapas[e.ID] = EntregaEtapa{Etapa: string(e.Etapa), Aprobado: e.AprobadoPorID != nil}
	}
	grado := map[string]int{}
	aristas := []AristaGrafo{}
	for _, r := range relaciones {
		if !ids[r.OrigenID] || !ids[r.DestinoID] {
			continue // huerfana: el grafo la ignora, no rompe
		}
		aristas = append(aristas, AristaGrafo{ID: r.ID, Origen: r.OrigenID, Destino: r.DestinoID, Tipo: r.Tipo})
		grado[r.OrigenID]++
		grado[r.DestinoID]++
	}
	nodos := []NodoGrafo{}
	for _, e := range entregables {
		nodos = append(nodos, NodoGrafo{
			ID: e.ID, Nombre: e.Nombre, Etapa: etapas[e.ID].Etapa,
			Aprobado: etapas[e.ID].Aprobado, Grado: grado[e.ID],
		})
	}
	return GrafoRelaciones{Nodos: nodos, Aristas: aristas}
}

// EntregaEtapa: datos resumidos del documento para el nodo.
type EntregaEtapa struct {
	Etapa    string
	Aprobado bool
}

// VecinosDe: documentos directamente conectados a un documento (ambas
// direcciones), con el tipo de enlace. Para el panel de detalle de un nodo.
type Vecino struct {
	ID         string
	Nombre     string
	Tipo       TipoRelacion
	Direccion  string // "saliente" (el documento apunta al vecino) o "entrante"
	RelacionID string
}

func VecinosDe(documentoID string, entregables []Entregable, relaciones []RelacionDocumento) []Vecino {
	nombres := map[string]string{}
	for _, e := range entregables {
		nombres[e.ID] = e.Nombre
	}
	var out []Vecino
	for _, r := range relaciones {
		switch {
		case r.OrigenID == documentoID:
			out = append(out, Vecino{ID: r.DestinoID, Nombre: nombres[r.DestinoID], Tipo: r.Tipo, Direccion: "saliente", RelacionID: r.ID})
		case r.DestinoID == documentoID:
			out = append(out, Vecino{ID: r.OrigenID, Nombre: nombres[r.OrigenID], Tipo: r.Tipo, Direccion: "entrante", RelacionID: r.ID})
		}
	}
	return out
}

// CaminoEntre: BFS del camino mas corto entre dos documentos (trazabilidad:
// p. ej. "esta adenda deriva de que acuerdo"). nil si no hay camino.
func CaminoEntre(origenID, destinoID string, relaciones []RelacionDocumento) []string {
	if origenID == destinoID {
		return []string{origenID}
	}
	adyacencia := map[string][]string{}
	for _, r := range relaciones {
		adyacencia[r.OrigenID] = append(adyacencia[r.OrigenID], r.DestinoID)
	}
	visitados := map[string]bool{origenID: true}
	cola := [][]string{{origenID}}
	for len(cola) > 0 {
		camino := cola[0]
		cola = cola[1:]
		actual := camino[len(camino)-1]
		for _, sig := range adyacencia[actual] {
			if sig == destinoID {
				return append(camino, sig)
			}
			if !visitados[sig] {
				visitados[sig] = true
				nuevo := append(append([]string(nil), camino...), sig)
				cola = append(cola, nuevo)
			}
		}
	}
	return nil
}
