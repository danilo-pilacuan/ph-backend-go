# Plan de migracion del backend a Go

Fuente de verdad del alcance: `../docs/funcionalidades.md` y el codigo actual
en `../backend/app/`. Este documento registra la estimacion, el plan por fases
y la matriz de paridad contracto-a-contracto.

## 1. Dimension actual (Python/FastAPI)

| Archivo | Lineas | Rol |
| --- | --- | --- |
| `app/servicio.py` | 965 | Reglas de dominio sobre el repositorio |
| `app/api/rutas.py` | 450 | 21 rutas REST bajo /api |
| `app/almacen.py` | 270 | RepositorioMemoria + RepositorioPostgres (JSONB) |
| `app/semilla.py` | 302 | Datos semilla (ids y valores de stores/semilla.ts) |
| `app/domain/*` | ~640 | modelos, calculos, ciclo de vida, gates, matriz, dependencias |
| `app/auth.py` | 133 | PBKDF2 + JWT HS256 + ServicioCredenciales |
| `app/main.py` | 132 | Cableado FastAPI, CORS, handlers de error |
| `app/api/esquemas.py` | 64 | Cuerpos de peticion camelCase |
| `app/rate_limit.py` | 57 | Limitador de login en memoria |
| Tests (9 archivos) | ~1500 | 76 tests: dominio, API HTTP, persistencia |

Total aplicacion: ~2.600 lineas. Tests: 76 casos.

## 2. Requisitos de la migracion (acordados)

1. **Mismos contratos**: JSON camelCase identico (nombres de campo, valores de
   enums, codigos de error, mapeos HTTP). El frontend Angular no cambia.
2. **Misma base de datos**: PostgreSQL 17 con el esquema JSONB existente
   (tablas `usuarios`, `proyectos`, `etapas`, `entregables`, `bitacora`,
   `credenciales`). La semilla debe cargar y leer los mismos documentos.
3. **Nuevo repositorio**: `backend-go/` dentro de la raiz de ProjectHub,
   inicializado como repo git propio para registrarse como submodulo.

## 3. Decisiones tecnicas

- **Router HTTP**: `net/http` + Go 1.22+ (patrones `GET /api/proyectos/{id}`);
  sin framework para minimizar superficie y dependencias. CORS a mano
  (middleware propio de ~40 lineas).
- **JWT y PBKDF2**: implementados sin librerias externas (stdlib `crypto/*`)
  para paridad exacta de formato con `app/auth.py`; verificados contra los
  hashes legados de la tabla `credenciales`.
- **Serde camelCase**: tags JSON explicitos en los structs de `internal/domain`
  (no hay generacion de alias como Pydantic; los tags son el contrato).
- **Repositorio**: misma interfaz que `almacen.py` (getters, `*_agregar`,
  `*_reemplazar`); dos implementaciones: memoria y Postgres (`pgx` stdlib
  `database/sql` + driver `jackc/pgx`). El servicio no conoce SQL.
- **Estructura**:
  ```
  backend-go/
    cmd/server/main.go      # cableado, CORS, eleccion de repositorio
    internal/domain/        # modelos, reglas, errores (sin I/O)
    internal/almacen/       # memoria y Postgres (JSONB, mismas tablas)
    internal/auth/          # PBKDF2, JWT, credenciales
    internal/api/           # handlers HTTP, esquemas de entrada, rate limit
    internal/semilla/       # datos semilla identicos
  ```

## 4. Plan por fases

### Fase 0 - Esqueleto y dominio puro (completa)
- [x] Repo inicializado, go.mod, README, .gitignore, licencia de migracion.
- [x] `internal/domain`: catalogos, errores (codigos + mapeo HTTP), modelos con
      tags camelCase, matriz de permisos, ciclo de vida, gates, calculos,
      dependencias, PBKDF2/JWT sin dependencias.
- [x] Tests de dominio puro (espejo de test_dominio.py y test_dependencias.py):
      ciclo de vida, gates, calculos, matriz, dependencias y semilla.

### Fase 1 - Repositorios y semilla
- [x] `internal/almacen`: interfaz del repositorio (contrato de almacen.py).
- [x] RepositorioMemoria con semilla (identica a semilla.py, paridad JSON
      canonica verificada con golden test).
- [ ] RepositorioPostgres: esquema JSONB, semilla idempotente, upsert,
      reemplazo total; tabla `credenciales`.
- [ ] Tests: roundtrip Postgres (espejo de test_almacen.py).

### Fase 2 - Servicio de dominio (completa)
- [x] `internal/servicio`: casos de uso (proyectos, etapas, entregables,
      usuarios, gates, bitacora inmutable, re-sync total con todas sus
      validaciones VULN-001..008, ultimo Admin activo, unicidad de
      nombre/correo case-insensitive).
- [x] Tests espejo de test_dominio.py (reglas del servicio). Los casos de
      re-sync total se testean con la API (fase 3) y el golden dual (fase 4).

### Fase 3 - API HTTP
- [ ] `internal/api`: 21 rutas con los mismos metodos/paths/codigos de estado.
- [ ] Middleware: CORS (ALLOWED_ORIGINS), actor (Bearer o X-Actor-Id con los
      mismos 401/403/422), rate limit de login (429 LIMITE_INTENTOS).
- [ ] Body de error `{"codigo","mensaje"}` unificado.
- [ ] Tests httptest (espejo de test_api.py, test_auth.py, test_cors.py,
      test_rate_limit.py, test_ultimo_admin.py).

### Fase 4 - Paridad contracto-a-contracto
- [ ] Golden tests: same-request/dual-backend. Para cada endpoint, la respuesta
      JSON del backend Go y del Python deben ser byte-iguales (mismos ids de
      semilla, mismos formatos de fecha, mismo orden de colecciones).
- [ ] Migracion de datos: verificacion de que un snapshot volcado de la BD
      Python carga sin cambios en el backend Go y viceversa.
- [ ] docs: OpenAPI espejo (`/docs` de FastAPI no aplica; generar spec desde
      los handlers y publicarla en docs/).

### Fase 5 - Cutover
- [ ] Dockerfile multi-stage (distroless) + docker-compose equivalente.
- [ ] CI: build, vet, test, golden tests.
- [ ] Bifurcar puertos durante la transicion (8100 Python, 8101 Go) y comparar
      trafico real del frontend antes de apagar el Python.

## 5. Estimacion

Basada en ~2.600 lineas de aplicacion + 76 tests a replicar. Esfuerzo en
dias-hombre de 6 h efectivas; los rangos asumen una persona con Go intermedio.

| Fase | Alcance | Estimacion |
| --- | --- | --- |
| 0. Esqueleto y dominio puro | catalogos, errores, modelos, matriz, ciclo de vida, gates, calculos, dependencias, PBKDF2/JWT + tests | 2-3 d |
| 1. Repositorios y semilla | interfaz, memoria, Postgres JSONB, credenciales + tests roundtrip | 1,5-2 d |
| 2. Servicio de dominio | casos de uso + re-sync con validaciones VULN + ultimo admin + tests | 3-4 d |
| 3. API HTTP | 21 rutas, CORS, actor, rate limit, handlers de error + tests httptest | 2-3 d |
| 4. Paridad contracto-a-contracto | golden tests dual-backend, migracion de snapshot, OpenAPI | 1,5-2 d |
| 5. Cutover | Docker, CI, bifurcacion de puertos, apagado de Python | 1-2 d |
| **Total** | | **11-16 dias-hombre** |

Riesgos y colchon:

- **Re-sync total (13.4)** es la pieza mas densa: fusion de bitacora append-only,
  validaciones VULN-001..008, normalizacion de usuarios y prefijos de id.
  Colchon incluido en la fase 2; si el golden test lo desvia, es el punto donde
  mas probable que crezca la estimacion (+1-2 d).
- **Fechas como string ISO**: Go no tiene el tipo "date naive" de Python; el
  parseo strict `2006-01-02` y el redondeo a 2 decimales deben replicarse
  exactamente para que los calculos de avance/presupuesto den los mismos
  numeros en la misma entrada.
- **Orden de colecciones**: el Python serializa listas en orden de insercion
  (memoria) o `ORDER BY id` (Postgres). El Go debe respetar exactamente ese
  orden para que el golden test sea byte-igual; si hay diferencias se corrigen
  en la capa de repositorio, no en el frontend.
- **PBKDF2/JWT**: hecho a mano evita dependencias y garantiza la paridad de
  formato, pero exige los golden tests de hash y token de la fase 4 antes de
  any confiar en la coexistencia de ambos backends sobre la misma BD.

## 6. Definicion de terminado de la migracion

1. `go test ./...` verde y `go vet` limpio.
2. Golden tests dual-backend byte-iguales en todos los endpoints.
3. Un snapshot volcado de la BD Python carga sin cambios en el Go y produce
   las mismas respuestas REST.
4. El frontend Angular (shell + 4 remotes) opera sin cambio alguno: login con
   JWT, transiciones con gate, re-sync total del estado, edicion de usuarios.
5. `../backend/` marcado como deprecado (README apuntando aqui) y el submodulo
   `backend-go` registrado en la raiz de ProjectHub.
