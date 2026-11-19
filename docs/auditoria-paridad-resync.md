# Auditoria de paridad de seguridad del re-sync Go (VS-Go-001..008)

El espejo Go (`backend-go/internal/servicio/resync.go`) se escribio antes de
las mitigaciones VULN-005..011 de Python (`backend/app/servicio.py`). Esta
auditoria compara regla a regla y registra el estado de cada una. Los gaps se
cierran en RC5/dev con tests espejo de los pytest de Python.

| Id | Regla Python (VULN) | Estado inicial en Go | Estado final |
|---|---|---|---|
| VS-Go-001 | Etapas del push pasan por validar_etapa (fechas, avance 0-100, montos finitos >= 0) - VULN-005 | FALTA: etapas entrantes solo se validan por dependencias | CERRADO |
| VS-Go-002 | Proyectos del push con presupuesto estimado finito >= 0 - VULN-005 | FALTA | CERRADO |
| VS-Go-003 | Nombres de proyecto unicos case-insensitive en el push - VULN-007 | FALTA | CERRADO |
| VS-Go-004 | Referencias de etapa (proyectoId existente) y entregable (proyectoId existente) - VULN-006 | PARCIAL: solo responsable/creador/patrocinador en usuarios; proyectoId de etapa/entregable no se valida | CERRADO |
| VS-Go-005 | Autoaprobacion y fecha de aprobacion ISO de entregables - VULN-005/010 | FALTA: aprobadoPorId sin reglas | CERRADO |
| VS-Go-006 | Aprobador y creador de entregable deben existir (ademas del creadoPorID ya validado) - VULN-009 | PARCIAL: solo creador | CERRADO |
| VS-Go-007 | Entregable con nombre no vacio y version >= 1 - VULN-010 | FALTA | CERRADO |
| VS-Go-008 | ids duplicados dentro de cada coleccion del push | FALTA | CERRADO |
| VS-Go-009 | Entradas de bitacora NUEVAS: referencia a proyecto y autor existentes - VULN-006 | FALTA: validarEntradaBitacora solo chequea tipo/fecha | CERRADO |
| VS-Go-010 | Transicion de bitacora nueva exige etapaOrigen y etapaDestino - it66 | FALTA | CERRADO |
| VS-Go-011 | Regla de prefijos de id `^(prefijo-.+|prefijo\d+)$` (acepta semilla p1/e1/ent-1) - VULN-011 | DIFIERE: prefijoValido solo acepta prefijo- o prefijo+sin-guion (rechaza "ent-1" por el guion y colaria "p-x-y"); la forma semilla con guion (ent-1) es canonica | CERRADO (regex espejo) |
| VS-Go-012 | Diff de usuarios solo-Admin SOLO si el contenido cambia (re-sync sin diff de usuarios lo acepta cualquier actor, contrato 13.4) - VULN-001 parte segura | DIFIERE: Go exige Admin SIEMPRE (rompe el contrato de empuje post-mutacion del frontend no-Admin) | CERRADO (diff condicional) |

Notas de diseno alineadas con Python:
- El responsable de etapa y el aprobador de entregable deben EXISTIR pero
  pueden estar INACTIVOS (el canal normal permite desactivarlos sin
  reasignar; ese estado es alcanzable).
- El consumo mayor al asignado NO se rechaza (estado legitimo 5.4).
- La bitacora vigente no se revalida (historial inmutable 3.5, VULN-006).
- La credencial de usuarios huerfanas se limpia tras un push exitoso (14.5).
