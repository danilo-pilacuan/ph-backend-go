// Package servicio: re-sync total (6.6 / 13.4, espejo app/servicio.py). El
// espejo es transaccional: se valida TODO antes de mutar; en error no queda
// estado parcial. Fusion de bitacora append-only y VULN-001..008.
package servicio

import (
	"sort"
	"strings"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// EstadoCompleto: snapshot canonico (consulta de re-sync 6.6).
type EstadoCompleto struct {
	Usuarios    []domain.Usuario          `json:"usuarios"`
	Proyectos   []domain.Proyecto         `json:"proyectos"`
	Etapas      []domain.EtapaTrabajo     `json:"etapas"`
	Entregables []domain.Entregable       `json:"entregables"`
	Bitacora    []domain.BitacoraEntrada  `json:"bitacora"`
}

// EstadoCompleto: colecciones ordenadas por ID (determinismo para golden
// tests byte-iguales entre memoria y Postgres).
func (s *ServicioProyectos) EstadoCompleto() EstadoCompleto {
	usuarios := append([]domain.Usuario(nil), s.repo.Usuarios()...)
	proyectos := append([]domain.Proyecto(nil), s.repo.Proyectos()...)
	etapas := append([]domain.EtapaTrabajo(nil), s.repo.Etapas()...)
	entregables := append([]domain.Entregable(nil), s.repo.Entregables()...)
	bitacora := append([]domain.BitacoraEntrada(nil), s.repo.Bitacora()...)
	sort.Slice(usuarios, func(i, j int) bool { return usuarios[i].ID < usuarios[j].ID })
	sort.Slice(proyectos, func(i, j int) bool { return proyectos[i].ID < proyectos[j].ID })
	sort.Slice(etapas, func(i, j int) bool { return etapas[i].ID < etapas[j].ID })
	sort.Slice(entregables, func(i, j int) bool { return entregables[i].ID < entregables[j].ID })
	sort.Slice(bitacora, func(i, j int) bool { return bitacora[i].ID < bitacora[j].ID })
	return EstadoCompleto{Usuarios: usuarios, Proyectos: proyectos, Etapas: etapas, Entregables: entregables, Bitacora: bitacora}
}

// EscribirEstadoCompleto: espejo del entrante; valida y fusiona todo antes de
// tocar el repositorio local. actor = usuario autenticado del token.
func (s *ServicioProyectos) EscribirEstadoCompleto(entrante EstadoCompleto, actor domain.Usuario) error {
	if err := s.exigirActorValido(actor); err != nil {
		return err
	}
	if err := s.ExigirAdmin(actor); err != nil {
		return err
	}
	// VULN-001: diff de usuarios solo lo pide un Admin (ya exigido).
	usuarios := normalizarUsuarios(entrante.Usuarios)
	for i := range usuarios {
		u := &usuarios[i]
		if strings.TrimSpace(u.Nombre) == "" {
			return domain.ErrorValidacion("El nombre del usuario es obligatorio.")
		}
		if !correoValido(u.Correo) {
			return domain.ErrorValidacion("El correo no tiene un formato valido.")
		}
		if _, err := validarRoles(u.Roles); err != nil {
			return err
		}
	}
	if err := verificarCorreoUnico(usuarios); err != nil {
		return err
	}
	if err := verificarUltimoAdminEnEntrante(usuarios, s.repo.Usuarios()); err != nil {
		return err
	}
	idsUsuarios := map[string]bool{}
	for _, u := range usuarios {
		idsUsuarios[u.ID] = true
	}
	for _, p := range entrante.Proyectos {
		if err := prefijoValido(p.ID, "p"); err != nil {
			return err
		}
		if !idsUsuarios[p.PatrocinadorID] {
			return domain.ErrorValidacion("El patrocinador del proyecto %q no existe en el estado entrante.", p.Nombre)
		}
	}
	etapasPorProyecto := map[string][]domain.EtapaTrabajo{}
	for _, e := range entrante.Etapas {
		if err := prefijoValido(e.ID, "e"); err != nil {
			return err
		}
		if !idsUsuarios[e.ResponsableID] {
			return domain.ErrorValidacion("El responsable de la etapa %q no existe en el estado entrante.", e.Nombre)
		}
		etapasPorProyecto[e.ProyectoID] = append(etapasPorProyecto[e.ProyectoID], e)
	}
	for _, e := range entrante.Entregables {
		if err := prefijoValido(e.ID, "ent"); err != nil {
			return err
		}
		if !idsUsuarios[e.CreadoPorID] {
			return domain.ErrorValidacion("El autor del entregable %q no existe en el estado entrante.", e.Nombre)
		}
	}
	for pid, etapasDelProyecto := range etapasPorProyecto {
		for _, e := range etapasDelProyecto {
			if err := domain.ValidarDependencias(e.ID, pid, etapasDelProyecto); err != nil {
				return err
			}
		}
	}
	bitacoraFusionada, err := s.fusionarBitacora(s.repo.Bitacora(), entrante.Bitacora, actor.ID)
	if err != nil {
		return err
	}
	// Todo valido: aplicar (transaccion en memoria; en Postgres sera lote).
	s.repo.UsuariosReemplazar(usuarios)
	s.repo.ProyectosReemplazar(entrante.Proyectos)
	s.repo.EtapasReemplazar(entrante.Etapas)
	s.repo.EntregablesReemplazar(entrante.Entregables)
	s.repo.BitacoraReemplazar(bitacoraFusionada)
	return nil
}

// fusionarBitacora: append-only. Reglas 13.4: entrante modificado -> 422;
// entrada nueva con autor != actor -> 422; nuevas con prefijo "b-".
func (s *ServicioProyectos) fusionarBitacora(local, entrante []domain.BitacoraEntrada, actorID string) ([]domain.BitacoraEntrada, error) {
	localPorID := map[string]domain.BitacoraEntrada{}
	for _, b := range local {
		localPorID[b.ID] = b
	}
	fusion := append([]domain.BitacoraEntrada(nil), local...)
	for _, e := range entrante {
		if strings.HasPrefix(e.ID, "b-") {
			if previo, ok := localPorID[e.ID]; ok {
				if !bitacoraIgual(previo, e) {
					return nil, domain.ErrorValidacion("La entrada de bitacora %q es inmutable y no puede modificarse en el espejo.", e.ID)
				}
				continue
			}
			if e.AutorID != actorID {
				return nil, domain.ErrorValidacion("La entrada de bitacora %q es nueva y su autor %q no coincide con el actor %q.", e.ID, e.AutorID, actorID)
			}
			if err := validarEntradaBitacora(e); err != nil {
				return nil, err
			}
			fusion = append(fusion, e)
		}
	}
	sort.Slice(fusion, func(i, j int) bool { return fusion[i].ID < fusion[j].ID })
	return fusion, nil
}

func bitacoraIgual(a, b domain.BitacoraEntrada) bool {
	return a.Tipo == b.Tipo && a.ProyectoID == b.ProyectoID && a.AutorID == b.AutorID &&
		a.Fecha == b.Fecha && comentarioIgual(a.Comentario, b.Comentario) &&
		etapaPtrIgual(a.EtapaOrigen, b.EtapaOrigen) && etapaPtrIgual(a.EtapaDestino, b.EtapaDestino)
}

func comentarioIgual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func etapaPtrIgual(a, b *domain.EtapaCicloVida) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func validarEntradaBitacora(e domain.BitacoraEntrada) error {
	switch e.Tipo {
	case domain.TipoTransicion, domain.TipoAprobacion, domain.TipoCambio:
	default:
		return domain.ErrorValidacion("El tipo de bitacora %q no es valido.", string(e.Tipo))
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z07:00", e.Fecha); err != nil {
		if _, err2 := time.Parse("2006-01-02", e.Fecha); err2 != nil {
			return domain.ErrorValidacion("La fecha de la entrada de bitacora %q no es una fecha ISO valida.", e.ID)
		}
	}
	return nil
}

// normalizarUsuarios: nombre strip, correo lowercase (13.4).
func normalizarUsuarios(usuarios []domain.Usuario) []domain.Usuario {
	out := append([]domain.Usuario(nil), usuarios...)
	for i := range out {
		out[i].Nombre = strings.TrimSpace(out[i].Nombre)
		out[i].Correo = strings.ToLower(strings.TrimSpace(out[i].Correo))
	}
	return out
}

func verificarCorreoUnico(usuarios []domain.Usuario) error {
	vistos := map[string]bool{}
	for _, u := range usuarios {
		if vistos[u.Correo] {
			return domain.ErrorDuplicado("usuario", "correo", u.Correo)
		}
		vistos[u.Correo] = true
	}
	return nil
}

// verificarUltimoAdminEnEntrante: si al aplicar el entrante ya no quedaria un
// Admin activo (considerando el union entrante+local), rechaza.
func verificarUltimoAdminEnEntrante(entrante, local []domain.Usuario) error {
	localPorID := map[string]domain.Usuario{}
	for _, u := range local {
		localPorID[u.ID] = u
	}
	final := map[string]domain.Usuario{}
	for _, u := range local {
		final[u.ID] = u
	}
	for _, u := range entrante {
		final[u.ID] = u
	}
	for _, u := range final {
		if u.Activo && tieneRol(u.Roles, domain.RolAdmin) {
			return nil
		}
	}
	_ = localPorID
	return domain.ErrorValidacion("El espejo eliminaria el ultimo Admin activo del sistema.")
}

// prefijoValido: ids nuevos con prefijo u/p/e/ent o formato semilla u1/p1/e1/ent1.
func prefijoValido(id, prefijo string) error {
	if strings.HasPrefix(id, prefijo+"-") {
		return nil
	}
	if strings.HasPrefix(id, prefijo) && !strings.Contains(id, "-") {
		return nil
	}
	return domain.ErrorValidacion("El id %q no respeta el prefijo %q.", id, prefijo)
}
