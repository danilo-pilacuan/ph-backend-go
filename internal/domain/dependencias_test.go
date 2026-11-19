package domain

import (
	"reflect"
	"testing"
)

// ---------- Dependencias (6.5), espejo de test_dependencias.py ----------

func etapaDep(id, proyectoID string, dependencias []string, avance float64) EtapaTrabajo {
	return EtapaTrabajo{
		ID: id, ProyectoID: proyectoID, Nombre: "Etapa " + id,
		FechaInicio: "2026-01-01", FechaFin: "2026-01-31",
		ResponsableID: "u-pm", PresupuestoAsignado: 1000,
		Avance: avance, Dependencias: dependencias,
	}
}

func TestPrerequisitosSinYConDependencias(t *testing.T) {
	if len(etapaDep("e1", "p1", nil, 0).Dependencias) != 0 {
		t.Fatal("sin dependencias")
	}
	e := etapaDep("e1", "p1", []string{"e2", "e3"}, 0)
	if !reflect.DeepEqual(e.Dependencias, []string{"e2", "e3"}) {
		t.Fatalf("dependencias = %v", e.Dependencias)
	}
}

func TestDependientesDeEncuentraDependientes(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 0), etapaDep("e2", "p1", []string{"e1"}, 0), etapaDep("e3", "p1", []string{"e1"}, 0)}
	ids := []string{}
	for _, e := range DependientesDe("e1", etapas) {
		ids = append(ids, e.ID)
	}
	if !reflect.DeepEqual(ids, []string{"e2", "e3"}) {
		t.Fatalf("dependientes = %v", ids)
	}
}

func TestDependientesDeSinDependientes(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 0), etapaDep("e2", "p1", []string{"e1"}, 0)}
	if len(DependientesDe("e2", etapas)) != 0 {
		t.Fatal("e2 no tiene dependientes")
	}
}

func TestAutoDependenciaRechazada(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"e1"}, 0)}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("auto-dependencia debe rechazarse")
	}
}

func TestDuplicadoRechazado(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"e2", "e2"}, 0), etapaDep("e2", "p1", nil, 0)}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("prerequisito duplicado debe rechazarse")
	}
}

func TestPrerequisitoInexistenteRechazado(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"fantasma"}, 0)}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("prerequisito inexistente debe rechazarse")
	}
}

func TestPrerequisitoDeOtroProyectoRechazado(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"e9"}, 0), etapaDep("e9", "p2", nil, 0)}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("prerequisito de otro proyecto debe rechazarse")
	}
}

func TestCicloDosNodosRechazado(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"e2"}, 0), etapaDep("e2", "p1", []string{"e1"}, 0)}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("ciclo de dos nodos debe rechazarse")
	}
}

func TestCicloLargoRechazado(t *testing.T) {
	etapas := []EtapaTrabajo{
		etapaDep("e1", "p1", []string{"e3"}, 0),
		etapaDep("e2", "p1", []string{"e1"}, 0),
		etapaDep("e3", "p1", []string{"e2"}, 0),
	}
	if err := ValidarDependencias("e1", "p1", etapas); err == nil {
		t.Fatal("ciclo largo debe rechazarse")
	}
}

func TestCadenaValidaSinCiclo(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 0), etapaDep("e2", "p1", []string{"e1"}, 0), etapaDep("e3", "p1", []string{"e2"}, 0)}
	if err := ValidarDependencias("e3", "p1", etapas); err != nil {
		t.Fatalf("cadena valida debe pasar: %v", err)
	}
	if TieneCiclo("e3", etapas) {
		t.Fatal("cadena no tiene ciclo")
	}
}

func TestTieneCicloDetectaCiclo(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", []string{"e2"}, 0), etapaDep("e2", "p1", []string{"e1"}, 0)}
	if !TieneCiclo("e1", etapas) {
		t.Fatal("debe detectar el ciclo")
	}
}

func TestExigirDependenciasCompletasNoExigeConAvanceCero(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 50), etapaDep("e2", "p1", []string{"e1"}, 0)}
	if err := ExigirDependenciasCompletas(etapas[1], 0, etapas); err != nil {
		t.Fatalf("avance 0 no exige prerequisitos: %v", err)
	}
}

func TestExigirDependenciasCompletasLanzaConPrerequisitoIncompleto(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 50), etapaDep("e2", "p1", []string{"e1"}, 10)}
	err := ExigirDependenciasCompletas(etapas[1], 30, etapas)
	if err == nil {
		t.Fatal("prerequisito al 50% debe bloquear avance")
	}
}

func TestExigirDependenciasCompletasPasaConPrerequisitoCompleto(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 100), etapaDep("e2", "p1", []string{"e1"}, 0)}
	if err := ExigirDependenciasCompletas(etapas[1], 30, etapas); err != nil {
		t.Fatalf("prerequisito al 100%% debe permitir avance: %v", err)
	}
}

func TestDependenciasGanttIgnoraHuerfanas(t *testing.T) {
	etapas := []EtapaTrabajo{etapaDep("e1", "p1", nil, 0), etapaDep("e2", "p1", []string{"e1"}, 0), etapaDep("e3", "p1", []string{"x"}, 0)}
	aristas := DependenciasGantt(etapas)
	esperado := []map[string]string{{"prerequisitoId": "e1", "dependienteId": "e2"}}
	if len(aristas) != len(esperado) || aristas[0]["prerequisitoId"] != "e1" || aristas[0]["dependienteId"] != "e2" {
		t.Fatalf("aristas = %v", aristas)
	}
}
