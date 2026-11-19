package domain

import (
	"testing"
)

func entregableDe(id, proyectoID, nombre string, aprobado bool) Entregable {
	e := Entregable{ID: id, ProyectoID: proyectoID, Nombre: nombre, Etapa: EtapaPlanteamiento, Version: 1, CreadoPorID: "u-pm"}
	if aprobado {
		aprobo := "u-admin"
		e.AprobadoPorID = &aprobo
	}
	return e
}

func relacionDe(id, proyectoID, origen, destino string, tipo TipoRelacion) RelacionDocumento {
	return RelacionDocumento{ID: id, ProyectoID: proyectoID, OrigenID: origen, DestinoID: destino, Tipo: tipo, CreadoPorID: "u-pm", CreadoEn: "2026-11-19T00:00:00Z"}
}

func TestValidarRelacionTipoInvalido(t *testing.T) {
	rel := relacionDe("rel-1", "p1", "ent-1", "ent-2", "tipo-inventado")
	if err := ValidarRelacion(rel, nil, []Entregable{entregableDe("ent-1", "p1", "A", true), entregableDe("ent-2", "p1", "B", false)}); err == nil {
		t.Fatal("tipo inventado debio rechazarse")
	}
}

func TestValidarRelacionAutoenlaceRechazado(t *testing.T) {
	rel := relacionDe("rel-1", "p1", "ent-1", "ent-1", RelacionRelacionado)
	if err := ValidarRelacion(rel, nil, []Entregable{entregableDe("ent-1", "p1", "A", true)}); err == nil {
		t.Fatal("autoenlace debio rechazarse")
	}
}

func TestValidarRelacionParDuplicadoRechazado(t *testing.T) {
	existentes := []RelacionDocumento{relacionDe("rel-1", "p1", "ent-1", "ent-2", RelacionDerivaDe)}
	// Mismo par en la misma direccion.
	if err := ValidarRelacion(relacionDe("rel-2", "p1", "ent-1", "ent-2", RelacionAclara), existentes, nil); err == nil {
		t.Fatal("par duplicado en la misma direccion debio rechazarse")
	}
	// Mismo par en direccion inversa (el par solo puede enlazarse una vez).
	if err := ValidarRelacion(relacionDe("rel-3", "p1", "ent-2", "ent-1", RelacionAclara), existentes, nil); err == nil {
		t.Fatal("par duplicado en direccion inversa debio rechazarse")
	}
}

func TestValidarRelacionDocumentoFantasmaRechazado(t *testing.T) {
	docs := []Entregable{entregableDe("ent-1", "p1", "A", true)}
	if err := ValidarRelacion(relacionDe("rel-1", "p1", "ent-1", "ent-fantasma", RelacionModifica), nil, docs); err == nil {
		t.Fatal("destino fantasma debio rechazarse")
	}
	if err := ValidarRelacion(relacionDe("rel-1", "p1", "ent-fantasma", "ent-1", RelacionModifica), nil, docs); err == nil {
		t.Fatal("origen fantasma debio rechazarse")
	}
}

func TestValidarRelacionNotaMuyLargaRechazada(t *testing.T) {
	nota := make([]rune, 501)
	for i := range nota {
		nota[i] = 'a'
	}
	texto := string(nota)
	rel := relacionDe("rel-1", "p1", "ent-1", "ent-2", RelacionAclara)
	rel.Nota = &texto
	if err := ValidarRelacion(rel, nil, []Entregable{entregableDe("ent-1", "p1", "A", true), entregableDe("ent-2", "p1", "B", false)}); err == nil {
		t.Fatal("nota de 501 caracteres debio rechazarse")
	}
}

func TestValidarRelacionValidaPasa(t *testing.T) {
	docs := []Entregable{entregableDe("ent-1", "p1", "Contrato marco", true), entregableDe("ent-2", "p1", "Adenda 1", false)}
	rel := relacionDe("rel-1", "p1", "ent-2", "ent-1", RelacionModifica)
	if err := ValidarRelacion(rel, nil, docs); err != nil {
		t.Fatalf("relacion valida debio pasar: %v", err)
	}
}

func TestGrafoDeRelacionesGradosYAristas(t *testing.T) {
	docs := []Entregable{
		entregableDe("ent-1", "p1", "Contrato", true),
		entregableDe("ent-2", "p1", "Adenda 1", false),
		entregableDe("ent-3", "p1", "Acta de cierre", true),
	}
	relaciones := []RelacionDocumento{
		relacionDe("rel-1", "p1", "ent-2", "ent-1", RelacionModifica),
		relacionDe("rel-2", "p1", "ent-3", "ent-1", RelacionDerivaDe),
		relacionDe("rel-3", "p1", "ent-1", "ent-fantasma", RelacionRelacionado), // huerfana: ignorada
	}
	grafo := GrafoDeRelaciones(docs, relaciones)
	if len(grafo.Nodos) != 3 || len(grafo.Aristas) != 2 {
		t.Fatalf("grafo: %d nodos, %d aristas (huerfana ignorada)", len(grafo.Nodos), len(grafo.Aristas))
	}
	gradoDe := map[string]int{}
	for _, n := range grafo.Nodos {
		gradoDe[n.ID] = n.Grado
	}
	if gradoDe["ent-1"] != 2 || gradoDe["ent-2"] != 1 || gradoDe["ent-3"] != 1 {
		t.Fatalf("grados: %v", gradoDe)
	}
}

func TestVecinosDeAmbasDirecciones(t *testing.T) {
	docs := []Entregable{
		entregableDe("ent-1", "p1", "Contrato", true),
		entregableDe("ent-2", "p1", "Adenda 1", false),
		entregableDe("ent-3", "p1", "Acta", true),
	}
	relaciones := []RelacionDocumento{
		relacionDe("rel-1", "p1", "ent-2", "ent-1", RelacionModifica),
		relacionDe("rel-2", "p1", "ent-1", "ent-3", RelacionAclara),
	}
	vecinos := VecinosDe("ent-1", docs, relaciones)
	if len(vecinos) != 2 {
		t.Fatalf("vecinos de ent-1 = %d", len(vecinos))
	}
	// ent-2 -> ent-1 es entrante para ent-1; ent-1 -> ent-3 es saliente.
	vistos := map[string]string{}
	for _, v := range vecinos {
		vistos[v.ID] = v.Direccion
	}
	if vistos["ent-2"] != "entrante" || vistos["ent-3"] != "saliente" {
		t.Fatalf("direcciones: %v", vistos)
	}
}

func TestCaminoEntreTrazabilidad(t *testing.T) {
	relaciones := []RelacionDocumento{
		relacionDe("rel-1", "p1", "ent-2", "ent-1", RelacionModifica),
		relacionDe("rel-2", "p1", "ent-1", "ent-3", RelacionDerivaDe),
	}
	camino := CaminoEntre("ent-2", "ent-3", relaciones)
	esperado := []string{"ent-2", "ent-1", "ent-3"}
	if len(camino) != 3 {
		t.Fatalf("camino = %v", camino)
	}
	for i := range esperado {
		if camino[i] != esperado[i] {
			t.Fatalf("camino = %v, esperado %v", camino, esperado)
		}
	}
	// Sin camino.
	if camino := CaminoEntre("ent-3", "ent-2", relaciones); camino != nil {
		t.Fatalf("camino inverso no existe (grafo dirigido): %v", camino)
	}
	// Mismo documento.
	if camino := CaminoEntre("ent-1", "ent-1", relaciones); len(camino) != 1 || camino[0] != "ent-1" {
		t.Fatalf("camino a si mismo = %v", camino)
	}
}
