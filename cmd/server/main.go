// Punto de entrada del servidor Go (espejo de app/main.py). Elige
// repositorio por DATABASE_URL con fallback a memoria con semilla; credenciales
// sobre Postgres o memoria; AUTH_SECRET para firmar JWT. Puerto por PORT
// (8100 por defecto, el mismo que el backend Python).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/almacen"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/api"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/auth"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/servicio"
)

func main() {
	logger := log.New(os.Stdout, "projecthub-go ", log.LstdFlags)

	dsn := os.Getenv("DATABASE_URL")
	var repo almacen.Repositorio
	if dsn == "" {
		logger.Print("DATABASE_URL no definida: se usa RepositorioMemoria con semilla.")
		repo = almacen.NuevoMemoria(true)
	} else {
		pg, err := almacen.NuevoPostgres(dsn)
		if err != nil {
			logger.Printf("PostgreSQL no disponible (%v); se usa RepositorioMemoria.", err)
			repo = almacen.NuevoMemoria(true)
		} else {
			if err := pg.CrearEsquema(); err != nil {
				logger.Printf("No se pudo crear el esquema (%v); se usa memoria.", err)
				repo = almacen.NuevoMemoria(true)
			} else if err := pg.CargarSemilla(); err != nil {
				logger.Printf("No se pudo cargar la semilla (%v); se usa memoria.", err)
				repo = almacen.NuevoMemoria(true)
			} else {
				logger.Print("Persistencia PostgreSQL activa (DATABASE_URL).")
				repo = pg
			}
			// Solo cerrar si caemos a memoria; si es el repo activo vive con el proceso.
			if _, esPg := repo.(*almacen.RepositorioPostgres); !esPg {
				pg.Cerrar()
			}
		}
	}

	serv := servicio.NuevoServicio(repo)

	// Credenciales: Postgres si hay DSN activo; memoria en fallback.
	var cred *auth.ServicioCredenciales
	if pg, esPg := repo.(*almacen.RepositorioPostgres); esPg {
		cred = auth.NuevoCredenciales(pg)
	} else {
		cred = auth.NuevoCredenciales(nil)
	}
	serv.AsignarCredenciales(cred)

	secreto := auth.Secreto(logger)
	servidor := api.NuevoServidor(serv, cred, secreto)

	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8100"
	}
	logger.Printf("ProjectHub API Go escuchando en :%s", puerto)
	if err := http.ListenAndServe(":"+puerto, servidor.Handler()); err != nil {
		logger.Fatalf("servidor: %v", err)
	}
}
