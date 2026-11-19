package semilla

import (
	"reflect"
	"testing"
)

// Espejo del test de consistencia de semilla de test_dependencias.py:
// e2 <- e1, e5 <- e4, e1 al 100% y e5 sin avance.
func TestSemillaEsConsistenteConDependencias(t *testing.T) {
	porID := map[string]struct {
		dependencias []string
		avance       float64
	}{}
	for _, e := range EtapasSemilla {
		porID[e.ID] = struct {
			dependencias []string
			avance       float64
		}{e.Dependencias, e.Avance}
	}
	if !reflect.DeepEqual(porID["e2"].dependencias, []string{"e1"}) {
		t.Fatalf("e2.dependencias = %v", porID["e2"].dependencias)
	}
	if !reflect.DeepEqual(porID["e5"].dependencias, []string{"e4"}) {
		t.Fatalf("e5.dependencias = %v", porID["e5"].dependencias)
	}
	if porID["e1"].avance != 100 || porID["e2"].avance != 55 {
		t.Fatalf("avances semilla: e1=%v e2=%v", porID["e1"].avance, porID["e2"].avance)
	}
	if porID["e5"].avance != 0 {
		t.Fatalf("e5 debe estar sin avance, tiene %v", porID["e5"].avance)
	}
}

func TestSemillaIdsUnicos(t *testing.T) {
	vistos := map[string]bool{}
	for _, u := range UsuariosSemilla {
		if vistos[u.ID] {
			t.Fatalf("usuario duplicado %s", u.ID)
		}
		vistos[u.ID] = true
	}
	for _, p := range ProyectosSemilla {
		if vistos[p.ID] {
			t.Fatalf("proyecto duplicado %s", p.ID)
		}
		vistos[p.ID] = true
	}
	for _, e := range EtapasSemilla {
		if vistos[e.ID] {
			t.Fatalf("etapa duplicada %s", e.ID)
		}
		vistos[e.ID] = true
	}
	for _, b := range BitacoraSemilla {
		if vistos[b.ID] {
			t.Fatalf("bitacora duplicada %s", b.ID)
		}
		vistos[b.ID] = true
	}
}
