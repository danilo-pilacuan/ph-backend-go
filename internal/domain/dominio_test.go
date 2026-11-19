package domain

import (
	"reflect"
	"testing"
)

// ---------- Ciclo de vida ----------

func TestCadenaLinealDeNueveEtapas(t *testing.T) {
	siguiente, ok := SiguienteEtapa(EtapaPlanteamiento)
	if !ok || siguiente != EtapaIniciativa {
		t.Fatalf("Planteamiento -> %q (ok=%v)", siguiente, ok)
	}
	if _, ok := SiguienteEtapa(EtapaCierre); ok {
		t.Fatal("Cierre no debe tener etapa siguiente")
	}
}

func TestTransicionInvalidaSaltandoEtapas(t *testing.T) {
	err := ValidarTransicion(EtapaPlanteamiento, EtapaPlaneacion)
	if de, ok := err.(*ErrorDominio); !ok || de.Codigo != CodigoTransicionInvalida {
		t.Fatalf("se esperaba TRANSICION_INVALIDA, se obtuvo %v", err)
	}
}

func TestRetrocesoSoloPermitidoEnPoCYValidacionPoC(t *testing.T) {
	if r, ok := EtapaRetrocesoPermitido(EtapaPoC); !ok || r != EtapaIniciativa {
		t.Fatalf("PoC -> %q (ok=%v)", r, ok)
	}
	if r, ok := EtapaRetrocesoPermitido(EtapaValidacionPoC); !ok || r != EtapaPoC {
		t.Fatalf("ValidacionPoC -> %q (ok=%v)", r, ok)
	}
	if _, ok := EtapaRetrocesoPermitido(EtapaEjecucion); ok {
		t.Fatal("Ejecucion no permite retroceso")
	}
	if err := ValidarTransicion(EtapaEjecucion, EtapaPlaneacion); err == nil {
		t.Fatal("Ejecucion -> Planeacion debe ser invalida")
	}
}

func TestTransicionDesdeCierreEsInvalida(t *testing.T) {
	if EsTransicionValida(EtapaCierre, EtapaCierre) {
		t.Fatal("Cierre -> Cierre debe ser invalida")
	}
}

// ---------- Gates ----------

func entregable(nombre string, etapa EtapaCicloVida, creadoPor, aprobadoPor *string) Entregable {
	e := Entregable{ID: "ent-" + nombre, ProyectoID: "p1", Etapa: etapa, Nombre: nombre, Version: 1, CreadoPorID: *creadoPor}
	if aprobadoPor != nil {
		e.AprobadoPorID = aprobadoPor
		e.AprobadoEn = &[]string{"2026-01-01T00:00:00Z"}[0]
	}
	return e
}

func TestGateCerradoFaltaEntregable(t *testing.T) {
	gate := EvaluarGate(EtapaPlanteamiento, []Entregable{})
	if gate.Abierto {
		t.Fatal("gate debe estar cerrado sin entregables")
	}
	if !reflect.DeepEqual(gate.Faltantes, EntregablesObligatorios[EtapaPlanteamiento]) {
		t.Fatalf("faltantes = %v", gate.Faltantes)
	}
}

func TestGateAbiertoConEntregablesAprobados(t *testing.T) {
	pm := "u-pm"
	sp := "u-sponsor"
	e := entregable("Acta de planteamiento", EtapaPlanteamiento, &pm, &sp)
	gate := EvaluarGate(EtapaPlanteamiento, []Entregable{e})
	if !gate.Abierto || len(gate.Faltantes) != 0 || len(gate.PendientesAprobacion) != 0 {
		t.Fatalf("gate = %+v", gate)
	}
}

func TestGatePendienteDeAprobacion(t *testing.T) {
	pm := "u-pm"
	e := entregable("Acta de planteamiento", EtapaPlanteamiento, &pm, nil)
	gate := EvaluarGate(EtapaPlanteamiento, []Entregable{e})
	if gate.Abierto {
		t.Fatal("gate debe estar cerrado con entregable pendiente")
	}
	if !reflect.DeepEqual(gate.PendientesAprobacion, []string{"Acta de planteamiento"}) {
		t.Fatalf("pendientes = %v", gate.PendientesAprobacion)
	}
}

func TestAutorNoApruebaSuPropioEntregable(t *testing.T) {
	pm := "u-pm"
	e := entregable("Acta de planteamiento", EtapaPlanteamiento, &pm, nil)
	if de, ok := ValidarAprobacion(e, "u-pm").(*ErrorDominio); !ok || de.Codigo != CodigoValidacion {
		t.Fatalf("se esperaba VALIDACION, se obtuvo %v", de)
	}
}

func TestEntregableAprobadoNoSeReaprueba(t *testing.T) {
	pm := "u-pm"
	sp := "u-sponsor"
	e := entregable("Acta de planteamiento", EtapaPlanteamiento, &pm, &sp)
	if de, ok := ValidarAprobacion(e, "u-admin").(*ErrorDominio); !ok || de.Codigo != CodigoValidacion {
		t.Fatalf("se esperaba VALIDACION, se obtuvo %v", de)
	}
}

// ---------- Calculos ----------

func TestDuracionMinimaUnDia(t *testing.T) {
	d1, err := DuracionDias("2026-01-01", "2026-01-01")
	if err != nil || d1 != 1 {
		t.Fatalf("duracion mismo dia = %d (err=%v)", d1, err)
	}
	d2, err := DuracionDias("2026-01-01", "2026-01-05")
	if err != nil || d2 != 5 {
		t.Fatalf("duracion 5 dias = %d (err=%v)", d2, err)
	}
}

func TestAvancePonderadoPorDuracion(t *testing.T) {
	etapas := []EtapaTrabajo{
		{FechaInicio: "2026-01-01", FechaFin: "2026-01-05", Avance: 100},
		{FechaInicio: "2026-01-01", FechaFin: "2026-01-05", Avance: 0},
	}
	avance, err := AvancePonderado(etapas)
	if err != nil || avance != 50.0 {
		t.Fatalf("avance = %v (err=%v)", avance, err)
	}
}

func TestPresupuestoTotalYExcedido(t *testing.T) {
	a := EtapaTrabajo{FechaInicio: "2026-01-01", FechaFin: "2026-01-02", PresupuestoAsignado: 100, PresupuestoConsumido: 100}
	b := EtapaTrabajo{FechaInicio: "2026-01-01", FechaFin: "2026-01-02", PresupuestoAsignado: 50, PresupuestoConsumido: 20}
	total, err := PresupuestoTotal([]EtapaTrabajo{a, b})
	if err != nil || total != 150.0 {
		t.Fatalf("total = %v (err=%v)", total, err)
	}
	if PresupuestoExcedido(a) {
		t.Fatal("consumo igual a asignado no esta excedido")
	}
	c := a
	c.PresupuestoConsumido = 101
	if !PresupuestoExcedido(c) {
		t.Fatal("consumo 101 sobre asignado 100 esta excedido")
	}
}

func TestValidarEtapaRechazaAvanceFueraDeRango(t *testing.T) {
	if err := ValidarEtapa("2026-01-01", "2026-01-02", 150, 0, 0); err == nil {
		t.Fatal("avance 150 debe rechazarse")
	}
	if err := ValidarEtapa("2026-01-01", "2026-01-02", -1, 0, 0); err == nil {
		t.Fatal("avance -1 debe rechazarse")
	}
}

func TestValidarEtapaRechazaFechasInvertidas(t *testing.T) {
	if err := ValidarEtapa("2026-02-01", "2026-01-01", 0, 0, 0); err == nil {
		t.Fatal("fechas invertidas deben rechazarse")
	}
}

func TestValidarEtapaRechazaMontoNegativo(t *testing.T) {
	if err := ValidarEtapa("2026-01-01", "2026-01-02", 0, -100, 0); err == nil {
		t.Fatal("presupuesto asignado negativo debe rechazarse")
	}
	if err := ValidarEtapa("2026-01-01", "2026-01-02", 0, 100, -5); err == nil {
		t.Fatal("presupuesto consumido negativo debe rechazarse")
	}
}

// ---------- Matriz de permisos ----------

func TestMatrizPermisosBasicos(t *testing.T) {
	if !TienePermiso(EtapaPlanteamiento, AccionEditarProyecto, []Rol{RolProjectManager}) {
		t.Fatal("PM puede editar proyecto en Planteamiento")
	}
	if TienePermiso(EtapaPlanteamiento, AccionEditarProyecto, []Rol{RolAnalista}) {
		t.Fatal("Analista no puede editar proyecto en Planteamiento")
	}
	if !TienePermiso(EtapaCierre, AccionCerrarProyecto, []Rol{RolAdmin}) {
		t.Fatal("Admin puede cerrar proyecto en Cierre")
	}
	if TienePermiso(EtapaCierre, AccionCerrarProyecto, []Rol{RolProjectManager}) {
		t.Fatal("PM no puede cerrar proyecto en Cierre")
	}
}

func TestCancelarYCerrarSonDeGobernanza(t *testing.T) {
	if !reflect.DeepEqual(Matriz[EtapaPlanteamiento][AccionCancelarProyecto], SoloGobernanza) {
		t.Fatalf("cancelarProyecto = %v", Matriz[EtapaPlanteamiento][AccionCancelarProyecto])
	}
	if !reflect.DeepEqual(Matriz[EtapaCierre][AccionCerrarProyecto], SoloGobernanza) {
		t.Fatalf("cerrarProyecto = %v", Matriz[EtapaCierre][AccionCerrarProyecto])
	}
	if TienePermiso(EtapaPlanteamiento, AccionCancelarProyecto, []Rol{RolProjectManager}) {
		t.Fatal("PM no puede cancelar proyecto")
	}
}
