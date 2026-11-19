package servicio

import (
	"testing"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

func nuevoServicioRelaciones(t *testing.T) (*ServicioProyectos, domain.Usuario) {
	t.Helper()
	s := NuevoServicio(almacen.NuevoMemoria(true))
	for _, u := range s.Usuarios() {
		if tieneRol(u.Roles, domain.RolAdmin) {
			return s, u
		}
	}
	t.Fatal("no hay Admin")
	return nil, domain.Usuario{}
}

func TestCrearRelacionYPersistencia(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionModifica
	creada, err := s.CrearRelacion("p1", DatosRelacion{
		OrigenID: dominioPtr("ent-7"), DestinoID: dominioPtr("ent-6"), Tipo: &tipo,
	}, admin)
	if err != nil {
		t.Fatalf("crear relacion: %v", err)
	}
	if creada.ID == "" || creada.CreadoPorID != admin.ID || creada.CreadoEn == "" {
		t.Fatalf("relacion creada incompleta: %+v", creada)
	}
	if len(s.RelacionesDeProyecto("p1")) != 1 {
		t.Fatal("la relacion debio persistir en el repo")
	}
	// Bitacora registra el enlace.
	hay := false
	for _, b := range s.BitacoraDeProyecto("p1") {
		if b.Tipo == domain.TipoCambio && b.Comentario != nil && *b.Comentario != "" {
			hay = true
		}
	}
	if !hay {
		t.Fatal("la creacion debio registrar entrada en bitacora")
	}
}

func dominioPtr(v string) *string { return &v }

func TestCrearRelacionDuplicadaRechazada(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionDerivaDe
	datos := DatosRelacion{OrigenID: dominioPtr("ent-2"), DestinoID: dominioPtr("ent-1"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p1", datos, admin); err != nil {
		t.Fatalf("primera creacion: %v", err)
	}
	if _, err := s.CrearRelacion("p1", datos, admin); err == nil {
		t.Fatal("par duplicado debio rechazarse (409)")
	}
	// Inversa tambien.
	inversa := DatosRelacion{OrigenID: dominioPtr("ent-1"), DestinoID: dominioPtr("ent-2"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p1", inversa, admin); err == nil {
		t.Fatal("inversa del par debio rechazarse (409)")
	}
}

func TestCrearRelacionAutoenlaceYFantasma(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionRelacionado
	auto := DatosRelacion{OrigenID: dominioPtr("ent-1"), DestinoID: dominioPtr("ent-1"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p1", auto, admin); err == nil {
		t.Fatal("autoenlace debio rechazarse")
	}
	fantasma := DatosRelacion{OrigenID: dominioPtr("ent-1"), DestinoID: dominioPtr("ent-fantasma"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p1", fantasma, admin); err == nil {
		t.Fatal("destino fantasma debio rechazarse (404)")
	}
	// Documento de otro proyecto.
	otro := DatosRelacion{OrigenID: dominioPtr("ent-1"), DestinoID: dominioPtr("ent-8"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p1", otro, admin); err == nil {
		t.Fatal("entregable de otro proyecto (ent-8 es de p3) debio rechazarse: no pertenece a p1")
	}
}

func TestCrearRelacionProyectoCanceladoBloqueado(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionAclara
	datos := DatosRelacion{OrigenID: dominioPtr("ent-1"), DestinoID: dominioPtr("ent-6"), Tipo: &tipo}
	if _, err := s.CrearRelacion("p6", datos, admin); err == nil {
		t.Fatal("p6 esta cancelado: no admite relacionar documentos")
	}
}

func TestEliminarRelacionSoloAdmin(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionAclara
	creada, err := s.CrearRelacion("p1", DatosRelacion{
		OrigenID: dominioPtr("ent-2"), DestinoID: dominioPtr("ent-1"), Tipo: &tipo,
	}, admin)
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	// Un PM no puede eliminar (solo Admin).
	var pm domain.Usuario
	for _, u := range s.Usuarios() {
		if tieneRol(u.Roles, domain.RolProjectManager) && !tieneRol(u.Roles, domain.RolAdmin) {
			pm = u
			break
		}
	}
	if err := s.EliminarRelacion(creada.ID, pm); err == nil {
		t.Fatal("un PM no puede eliminar relaciones")
	}
	if err := s.EliminarRelacion(creada.ID, admin); err != nil {
		t.Fatalf("eliminar por Admin: %v", err)
	}
	if len(s.RelacionesDeProyecto("p1")) != 0 {
		t.Fatal("la relacion debio eliminarse")
	}
	if err := s.EliminarRelacion(creada.ID, admin); err == nil {
		t.Fatal("eliminar de nuevo debio dar 404")
	}
}

func TestGrafoRelacionesViaServicio(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionModifica
	if _, err := s.CrearRelacion("p1", DatosRelacion{
		OrigenID: dominioPtr("ent-7"), DestinoID: dominioPtr("ent-6"), Tipo: &tipo,
	}, admin); err != nil {
		t.Fatalf("crear: %v", err)
	}
	grafo, err := s.GrafoRelaciones("p1")
	if err != nil {
		t.Fatalf("grafo: %v", err)
	}
	if len(grafo.Aristas) != 1 || grafo.Aristas[0].Tipo != domain.RelacionModifica {
		t.Fatalf("grafo: %+v", grafo)
	}
	gradoEnt6 := 0
	for _, n := range grafo.Nodos {
		if n.ID == "ent-6" {
			gradoEnt6 = n.Grado
		}
	}
	if gradoEnt6 != 1 {
		t.Fatalf("grado de ent-6 = %d", gradoEnt6)
	}
	if _, err := s.GrafoRelaciones("p-fantasma"); err == nil {
		t.Fatal("grafo de proyecto fantasma debio dar 404")
	}
}

func TestResyncConRelacionesOpcional(t *testing.T) {
	// Compatibilidad: un push SIN la coleccion relaciones conserva las existentes.
	s, admin := nuevoServicioRelaciones(t)
	tipo := domain.RelacionAclara
	if _, err := s.CrearRelacion("p1", DatosRelacion{
		OrigenID: dominioPtr("ent-2"), DestinoID: dominioPtr("ent-1"), Tipo: &tipo,
	}, admin); err != nil {
		t.Fatalf("crear: %v", err)
	}
	estado := semillaComoEstado(s)
	estado.Relaciones = nil // cliente que no conoce la coleccion
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("push sin relaciones debio pasar: %v", err)
	}
	if len(s.RelacionesDeProyecto("p1")) != 1 {
		t.Fatal("el push sin la coleccion debio conservar las relaciones existentes")
	}
	// Push CON relaciones: reemplaza (catalogo corregible).
	estado.Relaciones = []domain.RelacionDocumento{
		{ID: "rel-nueva-1", ProyectoID: "p1", OrigenID: "ent-7", DestinoID: "ent-6",
			Tipo: domain.RelacionDerivaDe, CreadoPorID: admin.ID, CreadoEn: "2026-11-19T00:00:00Z"},
	}
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("push con relaciones debio pasar: %v", err)
	}
	lista := s.RelacionesDeProyecto("p1")
	if len(lista) != 1 || lista[0].ID != "rel-nueva-1" {
		t.Fatalf("push debio reemplazar el catalogo: %v", lista)
	}
}

func TestResyncRelacionesInvalidasRechazadas(t *testing.T) {
	s, admin := nuevoServicioRelaciones(t)
	estado := semillaComoEstado(s)
	casos := []struct {
		nombre string
		mutar  func(r *domain.RelacionDocumento)
	}{
		{"tipo inventado", func(r *domain.RelacionDocumento) { r.Tipo = "inventado" }},
		{"autoenlace", func(r *domain.RelacionDocumento) { r.OrigenID = "ent-1"; r.DestinoID = "ent-1" }},
		{"proyecto fantasma", func(r *domain.RelacionDocumento) { r.ProyectoID = "p-fantasma" }},
		{"autor fantasma", func(r *domain.RelacionDocumento) { r.CreadoPorID = "u-fantasma" }},
		{"sin fecha ISO", func(r *domain.RelacionDocumento) { r.CreadoEn = "ayer" }},
		{"id sin prefijo", func(r *domain.RelacionDocumento) { r.ID = "x-relacion" }},
		{"documento fantasma", func(r *domain.RelacionDocumento) { r.OrigenID = "ent-fantasma" }},
	}
	for i, caso := range casos {
		rel := domain.RelacionDocumento{
			ID: "rel-nueva-" + string(rune('a'+i)), ProyectoID: "p1",
			OrigenID: "ent-7", DestinoID: "ent-6",
			Tipo: domain.RelacionModifica, CreadoPorID: admin.ID, CreadoEn: "2026-11-19T00:00:00Z",
		}
		caso.mutar(&rel)
		estado.Relaciones = []domain.RelacionDocumento{rel}
		if err := s.EscribirEstadoCompleto(estado, admin); err == nil {
			t.Fatalf("%s debio rechazarse (422)", caso.nombre)
		}
	}
}

func TestNormalizarNota(t *testing.T) {
	vacia := "   "
	if NormalizarNotaPublica(&vacia) != nil {
		t.Fatal("nota de espacios debio quedar nil")
	}
	texto := "  adenda que modifica clausula 3  "
	normalizada := NormalizarNotaPublica(&texto)
	if normalizada == nil || *normalizada != "adenda que modifica clausula 3" {
		t.Fatalf("nota normalizada = %v", normalizada)
	}
	if NormalizarNotaPublica(nil) != nil {
		t.Fatal("nota nil debio quedar nil")
	}
}
