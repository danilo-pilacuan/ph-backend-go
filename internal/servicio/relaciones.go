// Package servicio: relaciones entre documentos (16.1, ideas del dueno:
// relaciones tipo obsidian para un grafo de busqueda de acuerdos, adendas y
// relaciones). Los enlaces son parte del catalogo documental del proyecto:
// se validan contra los entregables existentes y se registran en bitacora.
package servicio

import (
	"sort"
	"strings"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// DatosRelacion: entrada de creacion (nota opcional).
type DatosRelacion struct {
	OrigenID  *string
	DestinoID *string
	Tipo      *domain.TipoRelacion
	Nota      *string
}

// RelacionesDeProyecto: enlaces del proyecto (consulta para tabla y grafo).
func (s *ServicioProyectos) RelacionesDeProyecto(proyectoID string) []domain.RelacionDocumento {
	var out []domain.RelacionDocumento
	for _, r := range s.repo.Relaciones() {
		if r.ProyectoID == proyectoID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Relaciones: todas (bootstrap plano 13.3).
func (s *ServicioProyectos) Relaciones() []domain.RelacionDocumento {
	lista := s.repo.Relaciones()
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista
}

// GrafoRelaciones: grafo del proyecto (nodos con grado + aristas tipadas).
func (s *ServicioProyectos) GrafoRelaciones(proyectoID string) (domain.GrafoRelaciones, error) {
	if s.ProyectoPorID(proyectoID) == nil {
		return domain.GrafoRelaciones{}, domain.ErrorEntidadNoEncontrada("Proyecto", proyectoID)
	}
	return domain.GrafoDeRelaciones(s.EntregablesDeProyecto(proyectoID), s.RelacionesDeProyecto(proyectoID)), nil
}

// VecinosDeDocumento: conexiones directas de un documento.
func (s *ServicioProyectos) VecinosDeDocumento(documentoID string) []domain.Vecino {
	return domain.VecinosDe(documentoID, s.repo.Entregables(), s.repo.Relaciones())
}

// CaminoRelacion: traza del camino mas corto entre dos documentos.
func (s *ServicioProyectos) CaminoRelacion(origenID, destinoID string) []string {
	return domain.CaminoEntre(origenID, destinoID, s.repo.Relaciones())
}

// CrearRelacion: valida invariante 16.1 (tipo del catalogo, no autoenlace,
// par unico, documentos existentes del proyecto), fija autor y fecha del
// servidor y registra en bitacora.
func (s *ServicioProyectos) CrearRelacion(proyectoID string, datos DatosRelacion, actor domain.Usuario) (domain.RelacionDocumento, error) {
	proyecto, err := s.exigirProyecto(proyectoID)
	if err != nil {
		return domain.RelacionDocumento{}, err
	}
	if err := s.exigirActorValido(actor); err != nil {
		return domain.RelacionDocumento{}, err
	}
	if err := s.exigirEstadoEditable(proyecto.Estado, "relacionar documentos"); err != nil {
		return domain.RelacionDocumento{}, err
	}
	if datos.OrigenID == nil || datos.DestinoID == nil || datos.Tipo == nil {
		return domain.RelacionDocumento{}, domain.ErrorValidacion("La relacion requiere origenId, destinoId y tipo.")
	}
	relacion := domain.RelacionDocumento{
		ID:          NuevoID("rel"),
		ProyectoID:  proyecto.ID,
		OrigenID:    *datos.OrigenID,
		DestinoID:   *datos.DestinoID,
		Tipo:        *datos.Tipo,
		CreadoPorID: actor.ID,
		CreadoEn:    time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Nota:        datos.Nota,
	}
	if err := domain.ValidarRelacion(relacion, s.repo.Relaciones(), s.EntregablesDeProyecto(proyectoID)); err != nil {
		return domain.RelacionDocumento{}, err
	}
	s.repo.RelacionesAgregar(relacion)
	comentario := "Relacion de documentos: \"" + nombreDe(s.repo.Entregables(), relacion.OrigenID) +
		"\" " + string(relacion.Tipo) + " \"" + nombreDe(s.repo.Entregables(), relacion.DestinoID) + "\"."
	s.registrarBitacora(proyecto.ID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return relacion, nil
}

// EliminarRelacion: solo Admin (el catalogo de enlaces es material de
// gobernanza); registra el retiro en bitacora.
func (s *ServicioProyectos) EliminarRelacion(relacionID string, actor domain.Usuario) error {
	if err := s.ExigirAdmin(actor); err != nil {
		return err
	}
	var objetivo *domain.RelacionDocumento
	lista := s.repo.Relaciones()
	for i := range lista {
		if lista[i].ID == relacionID {
			objetivo = &lista[i]
			break
		}
	}
	if objetivo == nil {
		return domain.ErrorEntidadNoEncontrada("Relacion de documentos", relacionID)
	}
	out := lista[:0]
	for _, r := range lista {
		if r.ID != relacionID {
			out = append(out, r)
		}
	}
	s.repo.RelacionesReemplazar(out)
	comentario := "Relacion de documentos retirada (" + string(objetivo.Tipo) + " " + objetivo.OrigenID + " -> " + objetivo.DestinoID + ")."
	s.registrarBitacora(objetivo.ProyectoID, actor.ID, domain.TipoCambio, nil, nil, &comentario)
	return nil
}

// normalizarNota: strip; nil si queda vacia. Exportada para la capa API.
func NormalizarNotaPublica(nota *string) *string {
	return normalizarNota(nota)
}

func nombreDe(entregables []domain.Entregable, id string) string {
	for _, e := range entregables {
		if e.ID == id {
			return e.Nombre
		}
	}
	return id
}

// normalizarNota: strip; nil si queda vacia.
func normalizarNota(nota *string) *string {
	if nota == nil {
		return nil
	}
	recortada := strings.TrimSpace(*nota)
	if recortada == "" {
		return nil
	}
	return &recortada
}
