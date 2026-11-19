package auth

import (
	"testing"
)

func TestCredencialesAutenticarSiembraDemo(t *testing.T) {
	repo := NuevoCredencialesMemoria()
	serv := NuevoCredenciales(repo)
	// Primer login: siembra la demo y verifica contra ella.
	if !serv.Autenticar("diana@projecthub.local", ClaveDemo()) {
		t.Fatal("la demo sembrada debe autenticar")
	}
	if serv.Autenticar("diana@projecthub.local", "mala") {
		t.Fatal("contrasena incorrecta no debe autenticar")
	}
	// La credencial sembrada quedo persistida en el repo.
	if _, ok := repo.Listar()["diana@projecthub.local"]; !ok {
		t.Fatal("la semilla perezosa debio guardar el hash")
	}
}

func TestCredencialesRegistrarYCambiar(t *testing.T) {
	serv := NuevoCredenciales(nil)
	serv.Registrar("paula@projecthub.local", "clave-inicial-1")
	if !serv.Autenticar("paula@projecthub.local", "clave-inicial-1") {
		t.Fatal("credencial registrada debe autenticar")
	}
	// Cambio con actual incorrecta: false, sin error.
	cambiada, err := serv.Cambiar("paula@projecthub.local", "mala", "clave-nueva-99")
	if cambiada || err != nil {
		t.Fatalf("cambio con actual mala: %v %v", cambiada, err)
	}
	// Cambio con nueva que viola la politica: ErrorValidacion.
	cambiada, err = serv.Cambiar("paula@projecthub.local", "clave-inicial-1", "corta")
	if cambiada || err == nil {
		t.Fatalf("nueva corta debio rechazarse: %v %v", cambiada, err)
	}
	// Cambio correcto.
	cambiada, err = serv.Cambiar("paula@projecthub.local", "clave-inicial-1", "clave-nueva-99")
	if !cambiada || err != nil {
		t.Fatalf("cambio valido: %v %v", cambiada, err)
	}
	if !serv.Autenticar("paula@projecthub.local", "clave-nueva-99") {
		t.Fatal("la nueva clave debe autenticar")
	}
	if serv.Autenticar("paula@projecthub.local", "clave-inicial-1") {
		t.Fatal("la vieja clave ya no debe autenticar")
	}
}

func TestCredencialesRenombrar(t *testing.T) {
	repo := NuevoCredencialesMemoria()
	serv := NuevoCredenciales(repo)
	serv.Registrar("vieja@projecthub.local", "clave-12345")
	hashViejo := repo.Listar()["vieja@projecthub.local"]
	serv.Renombrar("vieja@projecthub.local", "nueva@projecthub.local")
	hashes := repo.Listar()
	if hashes["nueva@projecthub.local"] != hashViejo {
		t.Fatal("el hash debe moverse al correo nuevo")
	}
	if _, ok := hashes["vieja@projecthub.local"]; ok {
		t.Fatal("el correo viejo debe eliminarse")
	}
	if !serv.Autenticar("nueva@projecthub.local", "clave-12345") {
		t.Fatal("la clave debe seguir verificando tras el renombre")
	}
	// Renombrar un correo inexistente: no-op sin error.
	serv.Renombrar("fantasma@projecthub.local", "otra@projecthub.local")
}

func TestCredencialesSincronizarConUsuarios(t *testing.T) {
	repo := NuevoCredencialesMemoria()
	serv := NuevoCredenciales(repo)
	serv.Registrar("a@projecthub.local", "clave-aaaa1")
	serv.Registrar("b@projecthub.local", "clave-bbbb1")
	serv.Registrar("c@projecthub.local", "clave-cccc1")
	// Solo a y b siguen existiendo como usuarios.
	serv.SincronizarConUsuarios(map[string]bool{"a@projecthub.local": true, "b@projecthub.local": true})
	hashes := repo.Listar()
	if len(hashes) != 2 {
		t.Fatalf("quedan %d credenciales, esperadas 2", len(hashes))
	}
	if _, ok := hashes["c@projecthub.local"]; ok {
		t.Fatal("la credencial huerfana debio eliminarse")
	}
}

func TestRepositorioCredencialesMemoriaContrato(t *testing.T) {
	repo := NuevoCredencialesMemoria()
	if err := repo.Guardar("x@projecthub.local", "hash-12345"); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if repo.Listar()["x@projecthub.local"] != "hash-12345" {
		t.Fatal("listar debe devolver el hash")
	}
	if err := repo.Eliminar("x@projecthub.local"); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if len(repo.Listar()) != 0 {
		t.Fatal("eliminar debe vaciar")
	}
}
