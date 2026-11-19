package almacen

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/semilla"
)

// DSN de prueba: configurable con TEST_DATABASE_URL; si no hay Postgres el
// paquete compila y los tests se saltan (mismo enfoque que test_almacen.py).
func dsnPrueba(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:test@localhost:55432/postgres?sslmode=disable"
	}
	repo, err := NuevoPostgres(dsn)
	if err != nil {
		t.Skipf("Postgres de prueba no disponible (%s): %v", dsn, err)
	}
	t.Cleanup(func() { repo.Cerrar() })
	return dsn
}

func nuevoPostgresLimpio(t *testing.T) *RepositorioPostgres {
	t.Helper()
	dsn := dsnPrueba(t)
	repo, err := NuevoPostgres(dsn)
	if err != nil {
		t.Skipf("Postgres de prueba no disponible: %v", err)
	}
	t.Cleanup(func() { repo.Cerrar() })
	if err := repo.CrearEsquema(); err != nil {
		t.Fatalf("crear esquema: %v", err)
	}
	// Limpieza total de colecciones para partir de cero en cada test.
	for _, nombre := range []string{"usuarios", "proyectos", "etapas", "entregables", "bitacora", "credenciales"} {
		if _, err := repo.db.Exec(`DELETE FROM ` + fmt.Sprintf("%q", nombre)); err != nil {
			t.Fatalf("limpieza de %s: %v", nombre, err)
		}
	}
	return repo
}

func TestPostgresSemillaYConsultaOrdenada(t *testing.T) {
	repo := nuevoPostgresLimpio(t)
	if err := repo.CargarSemilla(); err != nil {
		t.Fatalf("cargar semilla: %v", err)
	}
	// Idempotente: segunda carga no duplica.
	if err := repo.CargarSemilla(); err != nil {
		t.Fatalf("cargar semilla de nuevo: %v", err)
	}
	usuarios := repo.Usuarios()
	if len(usuarios) != len(semilla.UsuariosSemilla) {
		t.Fatalf("usuarios = %d, esperados %d", len(usuarios), len(semilla.UsuariosSemilla))
	}
	// Orden por id (paridad con Python ORDER BY id).
	ordenados := append([]domain.Usuario(nil), semilla.UsuariosSemilla...)
	for i := 1; i < len(ordenados); i++ {
		for j := i; j > 0 && ordenados[j].ID < ordenados[j-1].ID; j-- {
			ordenados[j], ordenados[j-1] = ordenados[j-1], ordenados[j]
		}
	}
	if !reflect.DeepEqual(usuarios, ordenados) {
		t.Fatal("los usuarios deben volver en ORDER BY id")
	}
	// Dependencias [] no null tras leer JSONB.
	for _, e := range repo.Etapas() {
		if e.Dependencias == nil {
			t.Fatalf("etapa %s volvio con dependencias null", e.ID)
		}
	}
}

func TestPostgresAgregarYReemplazar(t *testing.T) {
	repo := nuevoPostgresLimpio(t)
	if err := repo.CargarSemilla(); err != nil {
		t.Fatalf("cargar semilla: %v", err)
	}
	// Agregar un proyecto.
	nuevo := domain.Proyecto{ID: "p-nuevo", Nombre: "Proyecto roundtrip", Etapa: domain.EtapaPlanteamiento, Estado: domain.EstadoActivo}
	repo.ProyectosAgregar(nuevo)
	if len(repo.Proyectos()) != len(semilla.ProyectosSemilla)+1 {
		t.Fatal("agregar debe sumar un proyecto")
	}
	// Reemplazar todo: solo el nuevo.
	repo.ProyectosReemplazar([]domain.Proyecto{nuevo})
	quedan := repo.Proyectos()
	if len(quedan) != 1 || quedan[0].ID != "p-nuevo" {
		t.Fatalf("reemplazo total no aplico: %v", quedan)
	}
	// Upsert de etapa con dependencias [] serializadas (no null).
	etapa := domain.EtapaTrabajo{ID: "e-nuevo", ProyectoID: "p-nuevo", Dependencias: domain.DependenciasVacias()}
	repo.EtapasAgregar(etapa)
	leida := repo.Etapas()
	if len(leida) != len(semilla.EtapasSemilla)+1 {
		t.Fatalf("etapas = %d, esperadas %d", len(leida), len(semilla.EtapasSemilla)+1)
	}
	for _, e := range leida {
		if e.Dependencias == nil {
			t.Fatalf("etapa %s volvio con dependencias null", e.ID)
		}
	}
	// Bitacora reemplazo.
	repo.BitacoraReemplazar([]domain.BitacoraEntrada{{ID: "b-nuevo", ProyectoID: "p-nuevo", AutorID: "u-admin", Tipo: domain.TipoCambio}})
	if len(repo.Bitacora()) != 1 || repo.Bitacora()[0].ID != "b-nuevo" {
		t.Fatal("reemplazo de bitacora no aplico")
	}
}

func TestPostgresCredenciales(t *testing.T) {
	repo := nuevoPostgresLimpio(t)
	if err := repo.GuardarCredencial("a@x.local", "hash-a"); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if err := repo.GuardarCredencial("b@x.local", "hash-b"); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	// Upsert sobrescribe.
	if err := repo.GuardarCredencial("a@x.local", "hash-a2"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	creds, err := repo.Credenciales()
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if creds["a@x.local"] != "hash-a2" || creds["b@x.local"] != "hash-b" {
		t.Fatalf("credenciales = %v", creds)
	}
	if err := repo.EliminarCredencial("b@x.local"); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if err := repo.EliminarCredencialesExcepto([]string{"otra@x.local"}); err != nil {
		t.Fatalf("eliminar_excepto: %v", err)
	}
	creds, _ = repo.Credenciales()
	if len(creds) != 0 {
		t.Fatalf("quedan credenciales huerfanas: %v", creds)
	}
}
