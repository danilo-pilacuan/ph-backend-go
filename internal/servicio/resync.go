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
// Relaciones es opcional en el push (nil = sin cambios) para no romper a los
// clientes que aun no conocen la coleccion (16.1).
type EstadoCompleto struct {
	Usuarios    []domain.Usuario           `json:"usuarios"`
	Proyectos   []domain.Proyecto          `json:"proyectos"`
	Etapas      []domain.EtapaTrabajo      `json:"etapas"`
	Entregables []domain.Entregable        `json:"entregables"`
	Bitacora    []domain.BitacoraEntrada   `json:"bitacora"`
	Relaciones  []domain.RelacionDocumento `json:"relaciones,omitempty"`
}

// EstadoCompleto: colecciones ordenadas por ID (determinismo para golden
// tests byte-iguales entre memoria y Postgres).
func (s *ServicioProyectos) EstadoCompleto() EstadoCompleto {
	usuarios := append([]domain.Usuario(nil), s.repo.Usuarios()...)
	proyectos := append([]domain.Proyecto(nil), s.repo.Proyectos()...)
	etapas := append([]domain.EtapaTrabajo(nil), s.repo.Etapas()...)
	entregables := append([]domain.Entregable(nil), s.repo.Entregables()...)
	bitacora := append([]domain.BitacoraEntrada(nil), s.repo.Bitacora()...)
	relaciones := append([]domain.RelacionDocumento(nil), s.repo.Relaciones()...)
	sort.Slice(usuarios, func(i, j int) bool { return usuarios[i].ID < usuarios[j].ID })
	sort.Slice(proyectos, func(i, j int) bool { return proyectos[i].ID < proyectos[j].ID })
	sort.Slice(etapas, func(i, j int) bool { return etapas[i].ID < etapas[j].ID })
	sort.Slice(entregables, func(i, j int) bool { return entregables[i].ID < entregables[j].ID })
	sort.Slice(bitacora, func(i, j int) bool { return bitacora[i].ID < bitacora[j].ID })
	sort.Slice(relaciones, func(i, j int) bool { return relaciones[i].ID < relaciones[j].ID })
	return EstadoCompleto{Usuarios: usuarios, Proyectos: proyectos, Etapas: etapas, Entregables: entregables, Bitacora: bitacora, Relaciones: relaciones}
}

// EscribirEstadoCompleto: espejo del entrante; valida y fusiona todo antes de
// tocar el repositorio local. actor = usuario autenticado del token.
//
// Paridad de seguridad completa con app/servicio.py (VULN-001..011, ver
// docs/auditoria-paridad-resync.md): invariantes de dominio de etapas,
// proyectos y entregables, referencias, unicidad de ids y nombres, prefijos
// de id (dos formas canonicas), bitacora append-only con integridad de
// entradas nuevas y diff de usuarios solo-Admin condicionado a que cambie el
// contenido (contrato 13.4: un push sin diff de usuarios lo acepta cualquier
// actor autenticado).
func (s *ServicioProyectos) EscribirEstadoCompleto(entrante EstadoCompleto, actor domain.Usuario) error {
	if err := s.exigirActorValido(actor); err != nil {
		return err
	}
	if err := verificarIDsUnicos(entrante); err != nil {
		return err
	}
	usuarios := normalizarUsuarios(entrante.Usuarios)
	if err := s.validarUsuariosEntrantes(usuarios); err != nil {
		return err
	}
	if err := verificarUltimoAdminEnEntrante(usuarios, s.repo.Usuarios()); err != nil {
		return err
	}
	// VULN-001 parte segura: solo un Admin puede alterar la coleccion de
	// usuarios; un push sin diff de usuarios lo acepta cualquier actor.
	if err := s.validarDiffUsuarios(usuarios, actor); err != nil {
		return err
	}
	if err := validarInvariantesDominio(usuarios, entrante.Proyectos, entrante.Etapas, entrante.Entregables); err != nil {
		return err
	}
	idsProyectos := map[string]bool{}
	for _, p := range entrante.Proyectos {
		idsProyectos[p.ID] = true
	}
	idsUsuarios := map[string]bool{}
	for _, u := range usuarios {
		idsUsuarios[u.ID] = true
	}
	bitacoraFusionada, err := s.fusionarBitacora(s.repo.Bitacora(), entrante.Bitacora, actor.ID, idsProyectos, idsUsuarios)
	if err != nil {
		return err
	}
	// Relaciones (16.1): coleccion opcional. Un cliente que no la conoce no
	// la envia (nil): el servidor conserva la suya. Si viene, se valida la
	// invariante de cada enlace contra el estado entrante y se reemplaza
	// (los enlaces son catalogo corregible, no historial).
	relacionesActuales := s.repo.Relaciones()
	if entrante.Relaciones != nil {
		if err := validarRelacionesEntrantes(entrante.Relaciones, entrante.Proyectos, entrante.Entregables, usuarios); err != nil {
			return err
		}
		relacionesActuales = entrante.Relaciones
	}
	// Todo valido: aplicar (transaccion en memoria; en Postgres sera lote).
	s.repo.UsuariosReemplazar(usuarios)
	s.repo.ProyectosReemplazar(entrante.Proyectos)
	s.repo.EtapasReemplazar(entrante.Etapas)
	s.repo.EntregablesReemplazar(entrante.Entregables)
	s.repo.BitacoraReemplazar(bitacoraFusionada)
	s.repo.RelacionesReemplazar(relacionesActuales)
	// 14.5: limpieza de credenciales huerfanas tras el push.
	if s.credCambio != nil {
		correos := map[string]bool{}
		for _, u := range usuarios {
			correos[u.Correo] = true
		}
		s.credCambio.SincronizarConUsuarios(correos)
	}
	return nil
}

// verificarIDsUnicos: cada coleccion del push requiere ids string unicos.
func verificarIDsUnicos(entrante EstadoCompleto) error {
	grupos := [][]string{
		idsDe(entrante.Usuarios, func(u domain.Usuario) string { return u.ID }),
		idsDe(entrante.Proyectos, func(p domain.Proyecto) string { return p.ID }),
		idsDe(entrante.Etapas, func(e domain.EtapaTrabajo) string { return e.ID }),
		idsDe(entrante.Entregables, func(e domain.Entregable) string { return e.ID }),
		idsDe(entrante.Bitacora, func(b domain.BitacoraEntrada) string { return b.ID }),
	}
	if entrante.Relaciones != nil {
		grupos = append(grupos, idsDe(entrante.Relaciones, func(r domain.RelacionDocumento) string { return r.ID }))
	}
	for _, grupo := range grupos {
		vistos := map[string]bool{}
		for _, id := range grupo {
			if vistos[id] {
				return domain.ErrorValidacion("El estado entrante tiene ids duplicados en una coleccion (id %q).", id)
			}
			vistos[id] = true
		}
	}
	return nil
}

// validarRelacionesEntrantes (16.1): tipo del catalogo, no autoenlace, nota
// acotada, par unico y documentos existentes del proyecto entrante. Los
// enlaces son catalogo (no historial): el push reemplaza la coleccion.
func validarRelacionesEntrantes(relaciones []domain.RelacionDocumento, proyectos []domain.Proyecto, entregables []domain.Entregable, usuarios []domain.Usuario) error {
	idsProyectos := map[string]bool{}
	for _, p := range proyectos {
		idsProyectos[p.ID] = true
	}
	idsUsuarios := map[string]bool{}
	for _, u := range usuarios {
		idsUsuarios[u.ID] = true
	}
	for _, r := range relaciones {
		if err := prefijoValido(r.ID, "rel"); err != nil {
			return err
		}
		if !idsProyectos[r.ProyectoID] {
			return domain.ErrorValidacion("La relacion %q referencia al proyecto %q, que no existe.", r.ID, r.ProyectoID)
		}
		if !idsUsuarios[r.CreadoPorID] {
			return domain.ErrorValidacion("El autor %q de la relacion %q no existe.", r.CreadoPorID, r.ID)
		}
		if r.CreadoEn == "" || !fechaISOParseable(r.CreadoEn) {
			return domain.ErrorValidacion("La relacion %q requiere una fecha de creacion ISO valida.", r.ID)
		}
		delProyecto := entregablesDelProyectoDe(entregables, r.ProyectoID)
		if err := domain.ValidarRelacion(r, relaciones, delProyecto); err != nil {
			return err
		}
	}
	return nil
}

func entregablesDelProyectoDe(entregables []domain.Entregable, proyectoID string) []domain.Entregable {
	var out []domain.Entregable
	for _, e := range entregables {
		if e.ProyectoID == proyectoID {
			out = append(out, e)
		}
	}
	return out
}

func idsDe[T any](lista []T, clave func(T) string) []string {
	out := make([]string, 0, len(lista))
	for _, e := range lista {
		out = append(out, clave(e))
	}
	return out
}

// validarUsuariosEntrantes: espejo de _validar_nombre/_validar_roles/
// _validar_correo del canal normal (VULN-008); los usuarios ya llegan
// normalizados (nombre strip, correo lower).
func (s *ServicioProyectos) validarUsuariosEntrantes(usuarios []domain.Usuario) error {
	vistosCorreo := map[string]bool{}
	for _, u := range usuarios {
		if u.Nombre == "" {
			return domain.ErrorValidacion("El nombre del usuario es obligatorio.")
		}
		if len(u.Roles) == 0 {
			return domain.ErrorValidacion("El usuario requiere al menos un rol (1.4).")
		}
		if !correoValido(u.Correo) {
			return domain.ErrorValidacion("El correo no tiene un formato valido.")
		}
		if vistosCorreo[u.Correo] {
			return domain.ErrorDuplicado("usuario", "correo", u.Correo)
		}
		vistosCorreo[u.Correo] = true
	}
	return nil
}

// validarDiffUsuarios (VULN-001 parte segura): si el set de usuarios entrante
// difiere del vigente (nuevo, eliminado, roles, activo, correo o nombre),
// exige Admin. Un push identico lo acepta cualquier actor autenticado.
func (s *ServicioProyectos) validarDiffUsuarios(entrantes []domain.Usuario, actor domain.Usuario) error {
	if err := verificarUltimoAdminEnEntrante(entrantes, nil); err != nil {
		return err
	}
	vigentes := s.repo.Usuarios()
	if len(entrantes) == len(vigentes) {
		vigentesPorID := map[string]domain.Usuario{}
		for _, v := range vigentes {
			vigentesPorID[v.ID] = v
		}
		iguales := true
		for _, e := range entrantes {
			v, ok := vigentesPorID[e.ID]
			if !ok || !usuariosIguals(e, v) {
				iguales = false
				break
			}
		}
		if iguales {
			return nil
		}
	}
	if !tieneRol(actor.Roles, domain.RolAdmin) {
		return domain.ErrorPermisoDenegado("administrar usuarios", "sistema", []domain.Rol{domain.RolAdmin})
	}
	return nil
}

func usuariosIguals(a, b domain.Usuario) bool {
	if a.ID != b.ID || a.Nombre != b.Nombre || a.Correo != b.Correo || a.Activo != b.Activo || len(a.Roles) != len(b.Roles) {
		return false
	}
	for i := range a.Roles {
		if a.Roles[i] != b.Roles[i] {
			return false
		}
	}
	return true
}

// validarInvariantesDominio (espejo de _validar_invariantes_dominio de
// servicio.py, VULN-005/007/008/009/010/011): el push no es un canal para
// colar datos que la API normal rechaza.
func validarInvariantesDominio(usuarios []domain.Usuario, proyectos []domain.Proyecto, etapas []domain.EtapaTrabajo, entregables []domain.Entregable) error {
	idsUsuarios := map[string]bool{}
	for _, u := range usuarios {
		idsUsuarios[u.ID] = true
	}
	idsProyectos := map[string]bool{}
	for _, p := range proyectos {
		idsProyectos[p.ID] = true
	}
	// Etapas (VULN-005/006): proyecto y responsable existentes + validar_etapa
	// completa (fechas validas y coherentes, avance 0-100, montos finitos >= 0).
	// El responsable debe existir pero puede estar inactivo (el canal normal
	// permite desactivarlo sin reasignar etapas: estado alcanzable). El consumo
	// mayor al asignado NO se rechaza (estado legitimo 5.4).
	etapasPorProyecto := map[string][]domain.EtapaTrabajo{}
	for _, e := range etapas {
		if !idsProyectos[e.ProyectoID] {
			return domain.ErrorValidacion("La etapa %q referencia al proyecto %q, que no existe.", e.Nombre, e.ProyectoID)
		}
		if !idsUsuarios[e.ResponsableID] {
			return domain.ErrorValidacion("El responsable %q de la etapa %q no existe.", e.ResponsableID, e.Nombre)
		}
		if err := domain.ValidarEtapa(e.FechaInicio, e.FechaFin, e.Avance, e.PresupuestoAsignado, e.PresupuestoConsumido); err != nil {
			return err
		}
		etapasPorProyecto[e.ProyectoID] = append(etapasPorProyecto[e.ProyectoID], e)
	}
	// Proyectos (VULN-005/007): presupuesto finito >= 0 y nombre unico
	// case-insensitive (espejo de _existe_nombre).
	vistosNombre := map[string]bool{}
	for _, p := range proyectos {
		if !esMontoValido(p.PresupuestoEstimado) {
			return domain.ErrorValidacion("El presupuesto estimado no puede ser negativo.")
		}
		clave := strings.ToLower(p.Nombre)
		if vistosNombre[clave] {
			return domain.ErrorDuplicado("proyecto", "nombre", p.Nombre)
		}
		vistosNombre[clave] = true
	}
	// Entregables (VULN-005/009/010): proyecto existente, aprobacion valida
	// (fecha ISO, no autoaprobacion, aprobador existente), creador existente,
	// nombre no vacio y version >= 1.
	for _, ent := range entregables {
		if !idsProyectos[ent.ProyectoID] {
			return domain.ErrorValidacion("El entregable %q referencia al proyecto %q, que no existe.", ent.Nombre, ent.ProyectoID)
		}
		if ent.Nombre == "" {
			return domain.ErrorValidacion("El nombre del entregable es obligatorio.")
		}
		if ent.Version < 1 {
			return domain.ErrorValidacion("La version del entregable debe ser al menos 1.")
		}
		if !idsUsuarios[ent.CreadoPorID] {
			return domain.ErrorValidacion("El autor %q del entregable %q no existe.", ent.CreadoPorID, ent.Nombre)
		}
		if ent.AprobadoPorID == nil {
			continue
		}
		if ent.AprobadoEn == nil {
			return domain.ErrorValidacion("El entregable %q aprobado requiere fecha de aprobacion.", ent.Nombre)
		}
		if !fechaISOParseable(*ent.AprobadoEn) {
			return domain.ErrorValidacion("La fecha de aprobacion del entregable %q debe ser una fecha valida.", ent.Nombre)
		}
		if *ent.AprobadoPorID == ent.CreadoPorID {
			return domain.ErrorValidacion("El autor de un entregable no puede aprobar su propio documento.")
		}
		if !idsUsuarios[*ent.AprobadoPorID] {
			return domain.ErrorValidacion("El aprobador %q del entregable %q no existe.", *ent.AprobadoPorID, ent.Nombre)
		}
		// El aprobador puede estar inactivo: aprobar_entregable solo exige
		// actor activo en el momento de aprobar (1.8).
	}
	// Patrocinador del proyecto (VULN-009): debe existir entre los usuarios.
	for _, p := range proyectos {
		if !idsUsuarios[p.PatrocinadorID] {
			return domain.ErrorValidacion("El patrocinador %q del proyecto %q no existe.", p.PatrocinadorID, p.Nombre)
		}
	}
	// Prefijos de id (VULN-011): dos formas canonicas, prefijo-... (nuevo_id)
	// y prefijo+digitos (semilla p1/e1/b1/ent-1).
	for _, grupo := range []struct {
		prefijo string
		ids     []string
	}{
		{"u", idsDe(usuarios, func(u domain.Usuario) string { return u.ID })},
		{"p", idsDe(proyectos, func(p domain.Proyecto) string { return p.ID })},
		{"e", idsDe(etapas, func(e domain.EtapaTrabajo) string { return e.ID })},
		{"ent", idsDe(entregables, func(e domain.Entregable) string { return e.ID })},
	} {
		for _, id := range grupo.ids {
			if err := prefijoValido(id, grupo.prefijo); err != nil {
				return err
			}
		}
	}
	// Dependencias (6.5) por proyecto: igual que Python, solo se validan las
	// etapas CON dependencias y en el orden en que llegan del push (orden de
	// insercion del JSON). La deteccion de ciclos de Python (`siguiente in
	// pila` sobre la pila completa) es mas permisiva al recorrer desde las
	// raices declaradas; validar TODAS las etapas (incluidas las sin
	// dependencias) desde cualquier nodo rechazaria grafos que Python
	// acepta y bloquearia re-syncs legitimos del frontend.
	for _, e := range etapas {
		if len(e.Dependencias) == 0 {
			continue
		}
		if err := domain.ValidarDependencias(e.ID, e.ProyectoID, etapasPorProyecto[e.ProyectoID]); err != nil {
			return err
		}
	}
	return nil
}

// fechaISOParseable: acepta el formato del servidor y del frontend
// (RFC3339 con Z o offset, como datetime.fromisoformat con Z -> +00:00).
func fechaISOParseable(fecha string) bool {
	for _, formato := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02",
	} {
		if _, err := time.Parse(formato, fecha); err == nil {
			return true
		}
	}
	return false
}

// fusionarBitacora: append-only (3.5, VULN-001/006). Reglas: entrante que
// modifica una entrada vigente -> 422; entrada nueva con autor != actor ->
// 422; entrada nueva con referencia a proyecto/autor inexistente -> 422;
// fecha ISO y transicion completa en entradas nuevas; prefijo "b-" solo en
// entradas nuevas (el historial no se revalida: es inmutable).
func (s *ServicioProyectos) fusionarBitacora(local, entrante []domain.BitacoraEntrada, actorID string, idsProyectos, idsUsuarios map[string]bool) ([]domain.BitacoraEntrada, error) {
	localPorID := map[string]domain.BitacoraEntrada{}
	for _, b := range local {
		localPorID[b.ID] = b
	}
	fusion := append([]domain.BitacoraEntrada(nil), local...)
	for _, e := range entrante {
		previo, vigente := localPorID[e.ID]
		if vigente {
			if !bitacoraIgual(previo, e) {
				return nil, domain.ErrorValidacion("La bitacora es invariable: la entrada %q no puede modificarse en un re-sync (3.5).", e.ID)
			}
			continue
		}
		if e.AutorID != actorID {
			return nil, domain.ErrorValidacion("Solo puedes agregar entradas de bitacora propias: %q declara autorId %q y el actor es %q (VULN-001).", e.ID, e.AutorID, actorID)
		}
		if !idsProyectos[e.ProyectoID] {
			return nil, domain.ErrorValidacion("La entrada de bitacora %q referencia al proyecto %q, que no existe.", e.ID, e.ProyectoID)
		}
		if !idsUsuarios[e.AutorID] {
			return nil, domain.ErrorValidacion("La entrada de bitacora %q tiene un autor que no existe (3.5).", e.ID)
		}
		if err := validarEntradaBitacora(e); err != nil {
			return nil, err
		}
		fusion = append(fusion, e)
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
	if !fechaISOParseable(e.Fecha) {
		return domain.ErrorValidacion("La entrada de bitacora %q requiere una fecha ISO valida.", e.ID)
	}
	// Estructura de transiciones (it66): el canal normal siempre registra el
	// par de etapas; el push no puede colar transiciones sin origen/destino.
	if e.Tipo == domain.TipoTransicion && (e.EtapaOrigen == nil || e.EtapaDestino == nil) {
		return domain.ErrorValidacion("La transicion %q requiere etapa de origen y de destino.", e.ID)
	}
	// Prefijo de id de bitacora (it67): solo en entradas NUEVAS; el canal
	// normal siempre genera ids "b-...". El historial no se revalida (3.5).
	if !strings.HasPrefix(e.ID, "b-") {
		return domain.ErrorValidacion("El id %q de la coleccion %q debe comenzar con el prefijo %q.", e.ID, "bitacora", "b-")
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
	return domain.ErrorValidacion("El sistema debe conservar al menos un Admin activo; promueve a otro Admin antes de retirar el ultimo.")
}

// prefijoValido (VULN-011): dos formas canonicas, igual que Python:
// "prefijo-..." (ids de nuevo_id, p-ej p-20261119-1) y "prefijo+digitos"
// (semilla canonica p1, e1, b1, ent-1).
func prefijoValido(id, prefijo string) error {
	if strings.HasPrefix(id, prefijo+"-") {
		return nil
	}
	resto := strings.TrimPrefix(id, prefijo)
	if len(resto) > 0 && !strings.ContainsFunc(resto, func(r rune) bool { return r < '0' || r > '9' }) {
		return nil
	}
	return domain.ErrorValidacion("El id %q de la coleccion debe usar el prefijo %q- o el formato de semilla %q seguido de digitos.", id, prefijo, prefijo)
}
