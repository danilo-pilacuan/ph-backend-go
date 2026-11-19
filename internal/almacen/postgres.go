// Package almacen: persistencia en PostgreSQL 17 (DEC-006, espejo de
// RepositorioPostgres de app/almacen.py). Una tabla por coleccion con la
// entidad serializada a JSONB camelCase (el mismo contrato que la API REST).
// El esquema se crea al arrancar (CREATE TABLE IF NOT EXISTS); la semilla
// carga solo si la coleccion esta vacia (idempotente). Las listas se leen
// ORDER BY id, igual que la version Python.
package almacen

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/semilla"
)

type coleccionDef[T any] struct {
	nombre string
}

var coleccionUsuarios = coleccionDef[domain.Usuario]{"usuarios"}
var coleccionProyectos = coleccionDef[domain.Proyecto]{"proyectos"}
var coleccionEtapas = coleccionDef[domain.EtapaTrabajo]{"etapas"}
var coleccionEntregables = coleccionDef[domain.Entregable]{"entregables"}
var coleccionBitacora = coleccionDef[domain.BitacoraEntrada]{"bitacora"}

// RepositorioPostgres: misma interfaz Repositorio; JSONB por coleccion.
type RepositorioPostgres struct {
	db *sql.DB
}

// NuevoPostgres abre la conexion (driver pgx via database/sql).
func NuevoPostgres(dsn string) (*RepositorioPostgres, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &RepositorioPostgres{db: db}, nil
}

func (r *RepositorioPostgres) Cerrar() { _ = r.db.Close() }

// CrearEsquema: CREATE TABLE IF NOT EXISTS por coleccion (idempotente).
func (r *RepositorioPostgres) CrearEsquema() error {
	for _, nombre := range []string{"usuarios", "proyectos", "etapas", "entregables", "bitacora"} {
		if _, err := r.db.Exec(
			fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %q (id text PRIMARY KEY, datos jsonb NOT NULL)`, nombre)); err != nil {
			return err
		}
	}
	if _, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS "credenciales" (correo text PRIMARY KEY, hash text NOT NULL)`); err != nil {
		return err
	}
	return nil
}

// CargarSemilla: vuelca la semilla solo en colecciones vacias (idempotente),
// igual que cargar_semilla de Python.
func (r *RepositorioPostgres) CargarSemilla() error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	vacia := func(nombre string) (bool, error) {
		var n int
		if err := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %q`, nombre)).Scan(&n); err != nil {
			return false, err
		}
		return n == 0, nil
	}
	insertar := func(nombre, id string, entidad any) error {
		datos, err := json.Marshal(entidad)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			fmt.Sprintf(`INSERT INTO %q (id, datos) VALUES ($1, $2)`, nombre), id, datos)
		return err
	}
	if vaciaOK, err := vacia("usuarios"); err != nil {
		return err
	} else if vaciaOK {
		for _, u := range semilla.UsuariosSemilla {
			if err := insertar("usuarios", u.ID, u); err != nil {
				return err
			}
		}
	}
	if vaciaOK, err := vacia("proyectos"); err != nil {
		return err
	} else if vaciaOK {
		for _, p := range semilla.ProyectosSemilla {
			if err := insertar("proyectos", p.ID, p); err != nil {
				return err
			}
		}
	}
	if vaciaOK, err := vacia("etapas"); err != nil {
		return err
	} else if vaciaOK {
		for _, e := range semilla.EtapasSemilla {
			if err := insertar("etapas", e.ID, e); err != nil {
				return err
			}
		}
	}
	if vaciaOK, err := vacia("entregables"); err != nil {
		return err
	} else if vaciaOK {
		for _, e := range semilla.EntregablesSemilla {
			if err := insertar("entregables", e.ID, e); err != nil {
				return err
			}
		}
	}
	if vaciaOK, err := vacia("bitacora"); err != nil {
		return err
	} else if vaciaOK {
		for _, b := range semilla.BitacoraSemilla {
			if err := insertar("bitacora", b.ID, b); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ---------- Consultas (ORDER BY id, paridad con Python) ----------

func leerTodos[T any](r *RepositorioPostgres, nombre string) ([]T, error) {
	filas, err := r.db.Query(fmt.Sprintf(`SELECT datos FROM %q ORDER BY id`, nombre))
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []T{}
	for filas.Next() {
		var datos []byte
		var entidad T
		if err := filas.Scan(&datos); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(datos, &entidad); err != nil {
			return nil, err
		}
		if normalizador, ok := any(&entidad).(interface{ Normalizar() }); ok {
			normalizador.Normalizar()
		}
		out = append(out, entidad)
	}
	return out, filas.Err()
}

func (r *RepositorioPostgres) Usuarios() []domain.Usuario {
	l, err := leerTodos[domain.Usuario](r, "usuarios")
	if err != nil {
		log.Printf("ERROR leyendo usuarios: %v", err)
		return []domain.Usuario{}
	}
	return l
}

func (r *RepositorioPostgres) Proyectos() []domain.Proyecto {
	l, err := leerTodos[domain.Proyecto](r, "proyectos")
	if err != nil {
		log.Printf("ERROR leyendo proyectos: %v", err)
		return []domain.Proyecto{}
	}
	return l
}

func (r *RepositorioPostgres) Etapas() []domain.EtapaTrabajo {
	l, err := leerTodos[domain.EtapaTrabajo](r, "etapas")
	if err != nil {
		log.Printf("ERROR leyendo etapas: %v", err)
		return []domain.EtapaTrabajo{}
	}
	return l
}

func (r *RepositorioPostgres) Entregables() []domain.Entregable {
	l, err := leerTodos[domain.Entregable](r, "entregables")
	if err != nil {
		log.Printf("ERROR leyendo entregables: %v", err)
		return []domain.Entregable{}
	}
	return l
}

func (r *RepositorioPostgres) Bitacora() []domain.BitacoraEntrada {
	l, err := leerTodos[domain.BitacoraEntrada](r, "bitacora")
	if err != nil {
		log.Printf("ERROR leyendo bitacora: %v", err)
		return []domain.BitacoraEntrada{}
	}
	return l
}

// ---------- Escrituras ----------

func (r *RepositorioPostgres) escribirEntidad(nombre, id string, entidad any) error {
	datos, err := json.Marshal(entidad)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		fmt.Sprintf(`INSERT INTO %q (id, datos) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET datos = EXCLUDED.datos`, nombre),
		id, datos)
	return err
}

func (r *RepositorioPostgres) reemplazarTodo(nombre string, entidades []struct {
	ID      string
	Entidad any
}) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %q`, nombre)); err != nil {
		return err
	}
	for _, e := range entidades {
		datos, err := json.Marshal(e.Entidad)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %q (id, datos) VALUES ($1, $2)`, nombre), e.ID, datos); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *RepositorioPostgres) UsuariosReemplazar(l []domain.Usuario) {
	items := make([]struct {
		ID      string
		Entidad any
	}, 0, len(l))
	for _, u := range l {
		items = append(items, struct {
			ID      string
			Entidad any
		}{u.ID, u})
	}
	if err := r.reemplazarTodo("usuarios", items); err != nil {
		log.Printf("ERROR reemplazando usuarios: %v", err)
	}
}

func (r *RepositorioPostgres) ProyectosReemplazar(l []domain.Proyecto) {
	items := make([]struct {
		ID      string
		Entidad any
	}, 0, len(l))
	for _, p := range l {
		items = append(items, struct {
			ID      string
			Entidad any
		}{p.ID, p})
	}
	if err := r.reemplazarTodo("proyectos", items); err != nil {
		log.Printf("ERROR reemplazando proyectos: %v", err)
	}
}

func (r *RepositorioPostgres) EtapasReemplazar(l []domain.EtapaTrabajo) {
	items := make([]struct {
		ID      string
		Entidad any
	}, 0, len(l))
	for _, e := range l {
		items = append(items, struct {
			ID      string
			Entidad any
		}{e.ID, e})
	}
	if err := r.reemplazarTodo("etapas", items); err != nil {
		log.Printf("ERROR reemplazando etapas: %v", err)
	}
}

func (r *RepositorioPostgres) EntregablesReemplazar(l []domain.Entregable) {
	items := make([]struct {
		ID      string
		Entidad any
	}, 0, len(l))
	for _, e := range l {
		items = append(items, struct {
			ID      string
			Entidad any
		}{e.ID, e})
	}
	if err := r.reemplazarTodo("entregables", items); err != nil {
		log.Printf("ERROR reemplazando entregables: %v", err)
	}
}

func (r *RepositorioPostgres) BitacoraReemplazar(l []domain.BitacoraEntrada) {
	items := make([]struct {
		ID      string
		Entidad any
	}, 0, len(l))
	for _, b := range l {
		items = append(items, struct {
			ID      string
			Entidad any
		}{b.ID, b})
	}
	if err := r.reemplazarTodo("bitacora", items); err != nil {
		log.Printf("ERROR reemplazando bitacora: %v", err)
	}
}

func (r *RepositorioPostgres) UsuariosAgregar(u domain.Usuario) {
	_ = r.escribirEntidad("usuarios", u.ID, u)
}
func (r *RepositorioPostgres) ProyectosAgregar(p domain.Proyecto) {
	_ = r.escribirEntidad("proyectos", p.ID, p)
}
func (r *RepositorioPostgres) EtapasAgregar(e domain.EtapaTrabajo) {
	_ = r.escribirEntidad("etapas", e.ID, e)
}
func (r *RepositorioPostgres) EntregablesAgregar(e domain.Entregable) {
	_ = r.escribirEntidad("entregables", e.ID, e)
}
func (r *RepositorioPostgres) BitacoraAgregar(b domain.BitacoraEntrada) {
	_ = r.escribirEntidad("bitacora", b.ID, b)
}

var _ Repositorio = (*RepositorioPostgres)(nil)

// Eliminar borra por correo (interfaz auth.RepositorioCredenciales).
func (r *RepositorioPostgres) Eliminar(correo string) error { return r.EliminarCredencial(correo) }

// EliminarExcepto limpia huerfanas (interfaz auth.RepositorioCredenciales).
func (r *RepositorioPostgres) EliminarExcepto(correos []string) error {
	return r.EliminarCredencialesExcepto(correos)
}

// Listar devuelve el mapa correo -> hash (interfaz auth.RepositorioCredenciales).
func (r *RepositorioPostgres) Listar() map[string]string {
	m, err := r.Credenciales()
	if err != nil {
		log.Printf("ERROR leyendo credenciales: %v", err)
		return map[string]string{}
	}
	return m
}

// Guardar hace upsert del hash (interfaz auth.RepositorioCredenciales).
func (r *RepositorioPostgres) Guardar(correo, hash string) error {
	return r.GuardarCredencial(correo, hash)
}

// ---------- Credenciales (14.5, tabla propia) ----------

// GuardarCredencial: upsert correo -> hash PBKDF2.
func (r *RepositorioPostgres) GuardarCredencial(correo, hash string) error {
	_, err := r.db.Exec(
		`INSERT INTO "credenciales" (correo, hash) VALUES ($1, $2) ON CONFLICT (correo) DO UPDATE SET hash = EXCLUDED.hash`,
		correo, hash)
	return err
}

// EliminarCredencial borra por correo.
func (r *RepositorioPostgres) EliminarCredencial(correo string) error {
	_, err := r.db.Exec(`DELETE FROM "credenciales" WHERE correo = $1`, correo)
	return err
}

// EliminarCredencialesExcepto limpia huerfanas tras un re-sync total.
func (r *RepositorioPostgres) EliminarCredencialesExcepto(correos []string) error {
	if len(correos) == 0 {
		return nil
	}
	_, err := r.db.Exec(`DELETE FROM "credenciales" WHERE NOT (correo = ANY($1))`, correos)
	return err
}

// Credenciales: mapa correo -> hash.
func (r *RepositorioPostgres) Credenciales() (map[string]string, error) {
	filas, err := r.db.Query(`SELECT correo, hash FROM "credenciales" ORDER BY correo`)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := map[string]string{}
	for filas.Next() {
		var correo, hash string
		if err := filas.Scan(&correo, &hash); err != nil {
			return nil, err
		}
		out[correo] = hash
	}
	return out, filas.Err()
}
