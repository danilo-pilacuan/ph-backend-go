package servicio

import (
	"reflect"
	"testing"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

func nuevoServicio(t *testing.T) *ServicioProyectos {
	t.Helper()
	return NuevoServicio(almacen.NuevoMemoria(true))
}

// _usuario(rol): primer usuario de semilla con ese rol.
func usuarioRol(t *testing.T, s *ServicioProyectos, rol domain.Rol) domain.Usuario {
	t.Helper()
	for _, u := range s.Usuarios() {
		for _, r := range u.Roles {
			if r == rol {
				return u
			}
		}
	}
	t.Fatalf("la semilla no tiene usuario con rol %s", rol)
	return domain.Usuario{}
}

func inactivo(u domain.Usuario) domain.Usuario {
	u.Activo = false
	return u
}

// ---------- Reglas del servicio (test_dominio.py) ----------

func TestActorInactivoRechazado(t *testing.T) {
	s := nuevoServicio(t)
	admin := usuarioRol(t, s, domain.RolAdmin)
	nombre := "Proyecto nuevo"
	if _, err := s.CrearProyecto(DatosProyecto{Nombre: &nombre}, inactivo(admin)); err == nil {
		t.Fatal("un actor inactivo no puede crear proyectos")
	}
}

func TestNombreDuplicadoCaseInsensitive(t *testing.T) {
	s := nuevoServicio(t)
	pm := usuarioRol(t, s, domain.RolProjectManager)
	nombre := "PORTAL DE PAGOS" // semilla trae "Portal de pagos"
	_, err := s.CrearProyecto(DatosProyecto{Nombre: &nombre}, pm)
	if de, ok := err.(*domain.ErrorDominio); !ok || de.Codigo != domain.CodigoDuplicado {
		t.Fatalf("se esperaba DUPLICADO, se obtuvo %v", err)
	}
}

func TestNombreDuplicadoIgnoraElMismoProyectoAlEditar(t *testing.T) {
	s := nuevoServicio(t)
	pm := usuarioRol(t, s, domain.RolProjectManager)
	nombre := "Portal de PAGOS"
	if _, err := s.EditarProyecto("p1", DatosProyecto{Nombre: &nombre}, pm); err != nil {
		t.Fatalf("editar p1 con su propio nombre en otro case debe pasar: %v", err)
	}
}

func TestCrearProyectoExigePermiso(t *testing.T) {
	s := nuevoServicio(t)
	analista := usuarioRol(t, s, domain.RolAnalista)
	nombre := "Proyecto de analista"
	if _, err := s.CrearProyecto(DatosProyecto{Nombre: &nombre}, analista); err == nil {
		t.Fatal("un analista no puede crear proyectos")
	}
}

func TestProyectoCanceladoBloqueaEdicion(t *testing.T) {
	s := nuevoServicio(t)
	admin := usuarioRol(t, s, domain.RolAdmin)
	descripcion := "intento"
	if _, err := s.EditarProyecto("p6", DatosProyecto{Descripcion: &descripcion}, admin); err == nil {
		t.Fatal("no se puede editar un proyecto cancelado (p6)")
	}
}

func TestProyectoCanceladoBloqueaEtapas(t *testing.T) {
	s := nuevoServicio(t)
	pm := usuarioRol(t, s, domain.RolProjectManager)
	nombre := "Etapa nueva"
	inicio := "2026-01-01"
	fin := "2026-01-31"
	responsable := "u-pm"
	presupuesto := 1000.0
	_, err := s.CrearEtapa("p6", DatosEtapa{
		Nombre: &nombre, FechaInicio: &inicio, FechaFin: &fin,
		ResponsableID: &responsable, PresupuestoAsignado: &presupuesto,
	}, pm)
	if err == nil {
		t.Fatal("no se pueden crear etapas en un proyecto cancelado")
	}
}

func TestGateCerradoBloqueaTransicion(t *testing.T) {
	s := nuevoServicio(t)
	// p2 esta en Planeacion: le faltan entregables aprobados del gate.
	admin := usuarioRol(t, s, domain.RolAdmin)
	if _, err := s.Transicionar("p2", domain.EtapaContratos, admin, nil); err == nil {
		t.Fatal("el gate cerrado de p2 debe bloquear la transicion")
	}
}

func TestTransicionInvalidaSaltandoEtapa(t *testing.T) {
	s := nuevoServicio(t)
	admin := usuarioRol(t, s, domain.RolAdmin)
	if _, err := s.Transicionar("p1", domain.EtapaOperacion, admin, nil); err == nil {
		t.Fatal("saltar de Ejecucion a Operacion saltando etapas es invalido")
	}
}

func TestGateDeP1EjecucionBloqueaPorPendiente(t *testing.T) {
	s := nuevoServicio(t)
	// p1 esta en Ejecucion con ent-7 ("Acta de avance de ejecucion") pendiente.
	admin := usuarioRol(t, s, domain.RolAdmin)
	if _, err := s.Transicionar("p1", domain.EtapaOperacion, admin, nil); err == nil {
		t.Fatal("gate de Ejecucion cerrado: ent-7 pendiente de aprobacion")
	}
}

func TestResumenCalculaAvanceYPresupuesto(t *testing.T) {
	s := nuevoServicio(t)
	r, err := s.Resumen("p1")
	if err != nil {
		t.Fatalf("resumen p1: %v", err)
	}
	if r.PresupuestoTotal != 150000 {
		t.Fatalf("presupuesto total p1 = %v", r.PresupuestoTotal)
	}
	if r.ConsumoTotal != 81500 {
		t.Fatalf("consumo total p1 = %v", r.ConsumoTotal)
	}
	etapas := s.EtapasDeProyecto("p1")
	esperado, err := domain.AvancePonderado(etapas)
	if err != nil || r.Avance != esperado {
		t.Fatalf("avance p1 = %v (esperado %v, err=%v)", r.Avance, esperado, err)
	}
}

func TestGateActualReportaFaltantes(t *testing.T) {
	s := nuevoServicio(t)
	gate, err := s.GateActual("p2")
	if err != nil {
		t.Fatalf("gate p2: %v", err)
	}
	if gate.Abierto || len(gate.Faltantes) == 0 {
		t.Fatalf("gate de p2 en Planeacion debe estar cerrado con faltantes: %+v", gate)
	}
}

func TestCrearProyectoRegistraBitacora(t *testing.T) {
	s := nuevoServicio(t)
	pm := usuarioRol(t, s, domain.RolProjectManager)
	nombre := "Proyecto con bitacora"
	p, err := s.CrearProyecto(DatosProyecto{Nombre: &nombre}, pm)
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	hay := false
	for _, b := range s.Bitacora() {
		if b.ProyectoID == p.ID && b.Tipo == domain.TipoCambio && b.AutorID == pm.ID {
			hay = true
		}
	}
	if !hay {
		t.Fatal("la creacion debe registrar entrada en bitacora")
	}
}

func TestSemillaResumenIntacto(t *testing.T) {
	s := nuevoServicio(t)
	if !reflect.DeepEqual(len(s.Proyectos()), 6) || !reflect.DeepEqual(len(s.Etapas()), 5) {
		t.Fatalf("semilla: %d proyectos, %d etapas", len(s.Proyectos()), len(s.Etapas()))
	}
}
