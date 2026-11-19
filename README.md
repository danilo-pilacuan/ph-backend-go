# backend-go — ProjectHub API en Go

Reescritura del backend FastAPI (`../backend/`) en Go. Mismo dominio, mismos
contratos JSON camelCase, misma base de datos (PostgreSQL 17, JSONB por
coleccion). Este repositorio vive dentro de la raiz de ProjectHub y se registra
como submodulo git.

## Estado

Migracion en curso. Ver `docs/migracion.md` para el plan completo, la
estimacion y la matriz de paridad con la version Python.

## Ejecutar (cuando el server este completo)

```bash
go run ./cmd/server        # puerto 8100, igual que la API Python
go test ./...
```

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
