# backend-go — ProjectHub API en Go

Reescritura del backend FastAPI (`../backend/`) en Go. Mismo dominio, mismos
contratos JSON camelCase, misma base de datos (PostgreSQL 17, JSONB por
coleccion). Este repositorio vive dentro de la raiz de ProjectHub y se registra
como submodulo git.

## Estado

Migracion completa (fases 0-5 de `docs/migracion.md`): dominio puro, servicio,
repositorio memoria y Postgres JSONB, API HTTP de 21 rutas, paridad
contracto-a-contracto verificada contra el backend Python vivo y cutover
(Docker + CI). Pendiente solo operacional: registrar el submodulo en la raiz y
apagar el Python cuando se decida el cutover final.

## Ejecutar

```bash
go run ./cmd/server        # puerto 8100, igual que la API Python
go test ./...              # skipea tests Postgres si no hay BD en 55432
docker build -t projecthub-backend-go . && docker run -p 8100:8100 projecthub-backend-go
```

Variables: `DATABASE_URL` (Postgres JSONB; sin DSN usa memoria con semilla),
`AUTH_SECRET` (firma JWT; sin ella secreto efimero con warning),
`AUTH_CONTRASENA_DEMO` (default `projecthub-demo`), `ALLOWED_ORIGINS`,
`RATE_LIMIT_LOGIN_INTENTOS` (5), `RATE_LIMIT_LOGIN_VENTANA_SEG` (60), `PORT`
(8100).

## Convenciones

- Contrato JSON camelCase identico a `@projecthub/shared` y a la API Python
  (`patrocinadorId`, `presupuestoEstimado`, ...).
- Errores de dominio: body `{"codigo": "...", "mensaje": "..."}` con los
  mismos codigos y mapeos HTTP (404/409/422).
- Autenticacion: Bearer JWT HS256 (sub = id de usuario) o header
  `X-Actor-Id` como fallback; PBKDF2-HMAC-SHA256 240000 iteraciones,
  formato de hash compatible con `app/auth.py`.
- Persistencia: `DATABASE_URL` (Postgres JSONB, tablas `usuarios`,
  `proyectos`, `etapas`, `entregables`, `bitacora`, `credenciales`); sin DSN,
  repositorio en memoria con semilla identica a `stores/semilla.ts`.
