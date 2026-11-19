package servicio

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
)

// Golden de compatibilidad con el estado REAL de la demo: capturado por HTTP
// (GET /api/sincronizacion con Bearer) cuando el backend Go ya servia el
// stack. El push del propio estado debe pasar: es exactamente lo que hace el
// frontend tras cada mutacion (canal 13.4). Si esta prueba falla, el
// frontend no puede volver a empujar su estado y el re-sync queda bloqueado.
func TestPushEstadoRealDemoRoundtrip(t *testing.T) {
	brutos, err := os.ReadFile("testdata/estado_demo.json")
	if err != nil {
		t.Skipf("fixture no disponible: %v", err)
	}
	var estado EstadoCompleto
	if err := json.Unmarshal(brutos, &estado); err != nil {
		t.Fatalf("fixture invalido: %v", err)
	}
	// El vigente es el propio estado: el actor u-admin existe en el.
	s := NuevoServicio(almacen.NuevoMemoria(false))
	var admin domain.Usuario
	encontrado := false
	for _, u := range estado.Usuarios {
		if u.ID == "u-admin" {
			admin = u
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatal("el fixture debe contener al actor u-admin")
	}
	// El vigente inicial es el PROPIO estado demo (entradas de bitacora de
	// proyectos ya eliminados cargadas como historial vigente via canal
	// administrativo, igual que el demo real: bitacora_reemplazar fuera del
	// re-sync; VULN-006 conserva ese historial).
	s2 := NuevoServicio(almacen.NuevoMemoria(true))
	s.repo.BitacoraReemplazar(estado.Bitacora)
	// Cargar el vigente igual al entrante (push identico al estado).
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("push del estado real demo debio pasar: %v", err)
	}
	// Re-push del estado tras el primer push (roundtrip idempotente).
	if err := s.EscribirEstadoCompleto(estado, admin); err != nil {
		t.Fatalf("re-push identico debio pasar: %v", err)
	}
	// Push del estado real contra la semilla canonica como vigente (s2):
	// el estado trae entradas de bitacora de proyectos eliminados que son
	// NUEVAS para la semilla: Python las rechaza (VULN-006, integridad
	// referencial de entradas nuevas) y Go debe rechazarlas igual.
	if err := s2.EscribirEstadoCompleto(estado, admin); err == nil {
		t.Fatal("push con bitacora nueva referenciando proyectos eliminados debio rechazarse (paridad VULN-006)")
	}
}
