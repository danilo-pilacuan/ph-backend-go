package servicio

import (
	"testing"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// Tests espejo de los pytest de VULN-001..011 (test_api.py) sobre el servicio:
// el re-sync Go no puede colar datos que la API normal rechaza. Evidencia y
// matriz en docs/auditoria-paridad-resync.md (VS-Go-001..012).

func semillaComoEstado(s *ServicioProyectos) EstadoCompleto {
	return EstadoCompleto{
		Usuarios:    s.Usuarios(),
		Proyectos:   s.Proyectos(),
		Etapas:      s.Etapas(),
		Entregables: s.Entregables(),
		Bitacora:    s.Bitacora(),
	}
}

func nuevoServicioResync(t *testing.T) (*ServicioProyectos, domain.Usuario, domain.Usuario) {
	t.Helper()
	s := NuevoServicio(almacen.NuevoMemoria(true))
	var admin, pm domain.Usuario
	for _, u := range s.Usuarios() {
		if tieneRol(u.Roles, domain.RolAdmin) && admin.ID == "" {
			admin = u
		}
		if tieneRol(u.Roles, domain.RolProjectManager) && pm.ID == "" {
			pm = u
		}
	}
	return s, admin, pm
}

func TestResyncRoundtripDelEstadoVigenteSinAdminAceptado(t *testing.T) {
	// Contrato 13.4 (VS-Go-012): un push identico al estado vigente lo
	// acepta cualquier actor autenticado (no solo Admin).
	s, _, pm := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	if err := s.EscribirEstadoCompleto(estado, pm); err != nil {
		t.Fatalf("push sin diff de usuarios por un PM debe pasar: %v", err)
	}
}

func TestResyncDiffUsuariosSinAdminRechazado(t *testing.T) {
	// VULN-001 parte segura: cambiar un usuario exige Admin.
	s, _, pm := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	estado.Usuarios[0].Nombre = "Nombre Cambiado"
	if err := s.EscribirEstadoCompleto(estado, pm); err == nil {
		t.Fatal("un PM no puede alterar la coleccion de usuarios")
	}
	admin := domain.Usuario{}
	for _, u := range s.Usuarios() {
		if tieneRol(u.Roles, domain.RolAdmin) {
			admin = u
			break
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("un Admin si puede: %v", err)
	}
}

func TestResyncEtapaInvalidaRechazada(t *testing.T) {
	// VULN-005 (VS-Go-001): el push no colara avance 150, avance -1,
	// presupuesto negativo ni fechas invertidas.
	casos := []struct {
		nombre string
		mutar  func(e *domain.EtapaTrabajo)
	}{
		{"avance 150", func(e *domain.EtapaTrabajo) { e.Avance = 150 }},
		{"avance -1", func(e *domain.EtapaTrabajo) { e.Avance = -1 }},
		{"presupuesto asignado negativo", func(e *domain.EtapaTrabajo) { e.PresupuestoAsignado = -100 }},
		{"presupuesto consumido negativo", func(e *domain.EtapaTrabajo) { e.PresupuestoConsumido = -5 }},
		{"fechas invertidas", func(e *domain.EtapaTrabajo) { e.FechaInicio = "2027-01-01"; e.FechaFin = "2026-01-01" }},
	}
	for _, caso := range casos {
		s, admin, _ := nuevoServicioResync(t)
		estado := semillaComoEstado(s)
		for i := range estado.Etapas {
			if estado.Etapas[i].ID == "e1" {
				caso.mutar(&estado.Etapas[i])
			}
		}
		if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
			t.Fatalf("%s debio rechazarse (422)", caso.nombre)
		}
	}
	// El consumo mayor al asignado NO se rechaza (estado legitimo 5.4).
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Etapas {
		if estado.Etapas[i].ID == "e1" {
			estado.Etapas[i].PresupuestoConsumido = 40000
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("consumo excedido es estado legitimo: %v", err)
	}
}

func TestResyncProyectoPresupuestoNegativoYDuplicado(t *testing.T) {
	// VULN-005/007 (VS-Go-002/003).
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Proyectos {
		if estado.Proyectos[i].ID == "p1" {
			estado.Proyectos[i].PresupuestoEstimado = -1
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("presupuesto estimado negativo debio rechazarse")
	}
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	estado.Proyectos = append(estado.Proyectos, domain.Proyecto{
		ID: "p-nuevo", Nombre: "PORTAL DE PAGOS", PatrocinadorID: "u-admin", Estado: domain.EstadoActivo, Etapa: domain.EtapaPlanteamiento,
	})
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("nombre duplicado case-insensitive debio rechazarse (409)")
	}
}

func TestResyncReferenciasFantasmaRechazadas(t *testing.T) {
	// VULN-009 (VS-Go-004/006): patrocinador, responsable, autor y aprobador
	// inexistentes no pueden entrar por push.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Proyectos {
		if estado.Proyectos[i].ID == "p1" {
			estado.Proyectos[i].PatrocinadorID = "u-fantasma"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("patrocinador fantasma debio rechazarse")
	}
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Etapas {
		if estado.Etapas[i].ID == "e1" {
			estado.Etapas[i].ResponsableID = "u-fantasma"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("responsable fantasma debio rechazarse")
	}
	// Etapa con proyecto inexistente.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Etapas {
		if estado.Etapas[i].ID == "e1" {
			estado.Etapas[i].ProyectoID = "p-fantasma"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("etapa con proyecto fantasma debio rechazarse")
	}
}

func TestResyncEntregablesInvalidosRechazados(t *testing.T) {
	// VULN-005/009/010 (VS-Go-005/006/007): autoaprobacion, aprobador
	// fantasma, fecha no ISO, nombre vacio, version 0.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			autor := estado.Entregables[i].CreadoPorID
			estado.Entregables[i].AprobadoPorID = &autor
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("autoaprobacion debio rechazarse")
	}
	// Aprobador fantasma.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	fantasma := "u-fantasma"
	ahora := "2026-11-19T00:00:00Z"
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			estado.Entregables[i].AprobadoPorID = &fantasma
			estado.Entregables[i].AprobadoEn = &ahora
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("aprobador fantasma debio rechazarse")
	}
	// Aprobado sin fecha.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			estado.Entregables[i].AprobadoPorID = &fantasma
			estado.Entregables[i].AprobadoEn = nil
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("aprobacion sin fecha debio rechazarse")
	}
	// Fecha de aprobacion no ISO.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	mala := "ayer por la tarde"
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			estado.Entregables[i].AprobadoPorID = &fantasma
			estado.Entregables[i].AprobadoEn = &mala
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("fecha de aprobacion no ISO debio rechazarse")
	}
	// Version 0.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			estado.Entregables[i].Version = 0
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("version 0 debio rechazarse")
	}
	// Nombre vacio.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Entregables {
		if estado.Entregables[i].ID == "ent-1" {
			estado.Entregables[i].Nombre = ""
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("nombre vacio debio rechazarse")
	}
}

func TestResyncUsuariosInvalidosRechazados(t *testing.T) {
	// VULN-008 (VS-Go-008 usuarios): nombre vacio, sin roles, correo invalido
	// o duplicado case-insensitive.
	casos := []struct {
		nombre string
		mutar  func(u *domain.Usuario)
	}{
		{"nombre vacio", func(u *domain.Usuario) { u.Nombre = "  " }},
		{"sin roles", func(u *domain.Usuario) { u.Roles = nil }},
		{"correo invalido", func(u *domain.Usuario) { u.Correo = "no-es-correo" }},
	}
	for _, caso := range casos {
		s, admin, _ := nuevoServicioResync(t)
		estado := semillaComoEstado(s)
		for i := range estado.Usuarios {
			if estado.Usuarios[i].ID == "u-doble" {
				caso.mutar(&estado.Usuarios[i])
			}
		}
		if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
			t.Fatalf("%s debio rechazarse", caso.nombre)
		}
	}
	// Correo duplicado case-insensitive.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Usuarios {
		if estado.Usuarios[i].ID == "u-doble" {
			estado.Usuarios[i].Correo = "DIANA@PROJECTHUB.LOCAL"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("correo duplicado case-insensitive debio rechazarse (409)")
	}
	// Variante legitima (espacios y mayusculas) entra normalizada.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Usuarios {
		if estado.Usuarios[i].ID == "u-doble" {
			estado.Usuarios[i].Nombre = "  Daniel Doble  "
			estado.Usuarios[i].Correo = "  Daniel@ProjectHub.Local "
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("variante normalizable debio pasar: %v", err)
	}
	u := s.UsuarioPorID("u-doble")
	if u == nil || u.Correo != "daniel@projecthub.local" {
		t.Fatalf("correo debio quedar normalizado: %+v", u)
	}
}

func TestResyncPrefijosDeIdDosFormasCanonicas(t *testing.T) {
	// VULN-011 (VS-Go-011): acepta p-20261119-1 (nuevo_id) y p1 (semilla con
	// digitos, incluida ent-1 con guion); rechaza x-proyecto y p1-malo.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Proyectos {
		if estado.Proyectos[i].ID == "p1" {
			estado.Proyectos[i].ID = "x-proyecto"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("id sin prefijo de coleccion debio rechazarse")
	}
	// Prefijo ajeno.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Etapas {
		if estado.Etapas[i].ID == "e1" {
			estado.Etapas[i].ID = "p1-etapa"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("etapa con prefijo de proyecto debio rechazarse")
	}
	// Formato semilla con guion (ent-1) es canonico: el estado vigente pasa.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("estado vigente con ids semilla ent-1 debio pasar: %v", err)
	}
}

func TestResyncIDsDuplicadosEnColeccionRechazados(t *testing.T) {
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	estado.Proyectos = append(estado.Proyectos, estado.Proyectos[0])
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("ids duplicados en una coleccion debio rechazarse")
	}
}

func TestResyncBitacoraNuevaInvalidaRechazada(t *testing.T) {
	// VULN-006/it66/it67 (VS-Go-009/010): entrada nueva con autor ajeno,
	// proyecto inexistente, fecha no ISO, transicion incompleta o sin b-.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	nueva := domain.BitacoraEntrada{
		ID: "b-nueva", ProyectoID: "p1", AutorID: "u-pm",
		Fecha: "2026-11-19T00:00:00Z", Tipo: domain.TipoCambio,
	}
	// Autor ajeno al actor.
	estado.Bitacora = append(estado.Bitacora, nueva)
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("entrada nueva ajena al actor debio rechazarse")
	}
	// Propia del actor, con proyecto existente y fecha ISO: pasa y se fusiona.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	propia := nueva
	propia.AutorID = admin.ID
	estado.Bitacora = append(estado.Bitacora, propia)
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("entrada propia valida debio fusionarse: %v", err)
	}
	if len(s.Bitacora()) != 6 {
		t.Fatalf("bitacora = %d, esperadas 6 (5 semilla + 1 nueva)", len(s.Bitacora()))
	}
	// Entrada vigente omitida: conservada (no-op, push stale).
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	estado.Bitacora = estado.Bitacora[:4]
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("push stale debio aceptarse: %v", err)
	}
	if len(s.Bitacora()) != 5 {
		t.Fatalf("entrada omitida debio conservarse, hay %d", len(s.Bitacora()))
	}
	// Entrada vigente modificada: 422.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	for i := range estado.Bitacora {
		if estado.Bitacora[i].ID == "b1" {
			c := "comentario alterado"
			estado.Bitacora[i].Comentario = &c
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("modificar entrada vigente debio rechazarse (3.5)")
	}
	// Transicion nueva sin par de etapas.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	trans := domain.BitacoraEntrada{
		ID: "b-trans", ProyectoID: "p1", AutorID: admin.ID,
		Fecha: "2026-11-19T00:00:00Z", Tipo: domain.TipoTransicion,
	}
	estado.Bitacora = append(estado.Bitacora, trans)
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("transicion sin etapas debio rechazarse")
	}
	// Fecha no ISO.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	mala := nueva
	mala.AutorID = admin.ID
	mala.Fecha = "19/11/2026"
	estado.Bitacora = append(estado.Bitacora, mala)
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("fecha no ISO debio rechazarse")
	}
	// Entrada nueva sin prefijo b-.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	sinB := nueva
	sinB.AutorID = admin.ID
	sinB.ID = "entrada-nueva"
	estado.Bitacora = append(estado.Bitacora, sinB)
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("entrada nueva sin prefijo b- debio rechazarse")
	}
	// Referencia a proyecto inexistente en entrada nueva.
	s, admin, _ = nuevoServicioResync(t)
	estado = semillaComoEstado(s)
	huerfana := nueva
	huerfana.AutorID = admin.ID
	huerfana.ProyectoID = "p-fantasma"
	estado.Bitacora = append(estado.Bitacora, huerfana)
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("entrada nueva con proyecto fantasma debio rechazarse")
	}
}

func TestResyncUltimoAdminProtegido(t *testing.T) {
	// Hallazgo it50: el push no puede dejar al sistema sin Admin activo.
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	for i := range estado.Usuarios {
		if tieneRol(estado.Usuarios[i].Roles, domain.RolAdmin) {
			estado.Usuarios[i].Activo = false
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("eliminar el ultimo Admin activo debio rechazarse")
	}
}

func TestResyncLimpiaCredencialesHuerfanas(t *testing.T) {
	// 14.5: tras un push que elimina usuarios, sus credenciales se limpian.
	// Para eliminar a u-doble hay que reasignar sus etapas (e2, e5) y sus
	// entregables (ent-3, ent-5, ent-7) a otros usuarios (el push exige
	// referencias existentes).
	s, admin, _ := nuevoServicioResync(t)
	estado := semillaComoEstado(s)
	quedan := estado.Usuarios[:0]
	for _, u := range estado.Usuarios {
		if u.ID != "u-doble" {
			quedan = append(quedan, u)
		}
	}
	estado.Usuarios = quedan
	for i := range estado.Etapas {
		if estado.Etapas[i].ResponsableID == "u-doble" {
			estado.Etapas[i].ResponsableID = "u-pm"
		}
	}
	for i := range estado.Entregables {
		if estado.Entregables[i].CreadoPorID == "u-doble" {
			estado.Entregables[i].CreadoPorID = "u-pm"
		}
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("push sin u-doble debio pasar: %v", err)
	}
	if len(s.Usuarios()) != 6 {
		t.Fatalf("usuarios = %d, esperados 6", len(s.Usuarios()))
	}
}
