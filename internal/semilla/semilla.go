// Package semilla: datos semilla identicos a stores/semilla.ts y app/semilla.py.
// Mismos ids y valores que consume la UI hoy.
package semilla

import "github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"

func str(v string) *string                                 { return &v }
func etapa(v domain.EtapaCicloVida) *domain.EtapaCicloVida { return &v }

var UsuariosSemilla = []domain.Usuario{
	{ID: "u-admin", Nombre: "Diana Admin", Correo: "diana@projecthub.local", Roles: []domain.Rol{domain.RolAdmin}, Activo: true},
	{ID: "u-sponsor", Nombre: "Sergio Sponsor", Correo: "sergio@projecthub.local", Roles: []domain.Rol{domain.RolSponsor}, Activo: true},
	{ID: "u-pm", Nombre: "Paula Manager", Correo: "paula@projecthub.local", Roles: []domain.Rol{domain.RolProjectManager}, Activo: true},
	{ID: "u-analista", Nombre: "Andres Analista", Correo: "andres@projecthub.local", Roles: []domain.Rol{domain.RolAnalista}, Activo: true},
	{ID: "u-stakeholder", Nombre: "Sofia Stakeholder", Correo: "sofia@projecthub.local", Roles: []domain.Rol{domain.RolStakeholder}, Activo: true},
	{ID: "u-doble", Nombre: "Daniel Doble", Correo: "daniel@projecthub.local", Roles: []domain.Rol{domain.RolProjectManager, domain.RolAnalista}, Activo: true},
	{ID: "u-inactivo", Nombre: "Ivan Inactivo", Correo: "ivan@projecthub.local", Roles: []domain.Rol{domain.RolSponsor}, Activo: false},
}

var ProyectosSemilla = []domain.Proyecto{
	{ID: "p1", Nombre: "Portal de pagos", Descripcion: "Portal web de pagos en linea para clientes corporativos.", PatrocinadorID: "u-sponsor", PresupuestoEstimado: 180000, FechaObjetivo: "2027-03-31", Etapa: domain.EtapaEjecucion, Estado: domain.EstadoActivo},
	{ID: "p2", Nombre: "Data warehouse corporativo", Descripcion: "Modelo dimensional unico para reportes gerenciales.", PatrocinadorID: "u-sponsor", PresupuestoEstimado: 240000, FechaObjetivo: "2027-06-30", Etapa: domain.EtapaPlaneacion, Estado: domain.EstadoActivo},
	{ID: "p3", Nombre: "App movil de field service", Descripcion: "App para tecnicos en campo con trabajo offline.", PatrocinadorID: "u-admin", PresupuestoEstimado: 90000, FechaObjetivo: "2026-12-15", Etapa: domain.EtapaPoC, Estado: domain.EstadoActivo},
	{ID: "p4", Nombre: "Migracion ERP", Descripcion: "Migracion del ERP on-premise a la nube.", PatrocinadorID: "u-sponsor", PresupuestoEstimado: 320000, FechaObjetivo: "2026-11-30", Etapa: domain.EtapaCierre, Estado: domain.EstadoActivo},
	{ID: "p5", Nombre: "Chatbot de soporte", Descripcion: "Chatbot de primer nivel para mesa de ayuda.", PatrocinadorID: "u-admin", PresupuestoEstimado: 60000, FechaObjetivo: "2027-01-31", Etapa: domain.EtapaIniciativa, Estado: domain.EstadoActivo},
	{ID: "p6", Nombre: "Modernizacion legado", Descripcion: "Reemplazo del sistema legado de inventarios.", PatrocinadorID: "u-sponsor", PresupuestoEstimado: 150000, FechaObjetivo: "2026-10-31", Etapa: domain.EtapaEjecucion, Estado: domain.EstadoCancelado},
}

var EtapasSemilla = []domain.EtapaTrabajo{
	{ID: "e1", ProyectoID: "p1", Nombre: "Sprint 0 - cimientos", FechaInicio: "2026-09-01", FechaFin: "2026-09-30", ResponsableID: "u-pm", PresupuestoAsignado: 30000, PresupuestoConsumido: 28500, Avance: 100, Dependencias: domain.DependenciasVacias()},
	{ID: "e2", ProyectoID: "p1", Nombre: "Modulo de facturacion", FechaInicio: "2026-10-01", FechaFin: "2026-12-15", ResponsableID: "u-doble", PresupuestoAsignado: 70000, PresupuestoConsumido: 41000, Avance: 55, Dependencias: []string{"e1"}},
	{ID: "e3", ProyectoID: "p1", Nombre: "Integracion con pasarela", FechaInicio: "2026-11-01", FechaFin: "2027-02-28", ResponsableID: "u-pm", PresupuestoAsignado: 50000, PresupuestoConsumido: 12000, Avance: 20, Dependencias: domain.DependenciasVacias()},
	{ID: "e4", ProyectoID: "p2", Nombre: "Modelo dimensional", FechaInicio: "2026-11-10", FechaFin: "2027-01-31", ResponsableID: "u-analista", PresupuestoAsignado: 80000, PresupuestoConsumido: 0, Avance: 10, Dependencias: domain.DependenciasVacias()},
	{ID: "e5", ProyectoID: "p2", Nombre: "Pipeline de carga", FechaInicio: "2027-02-01", FechaFin: "2027-04-30", ResponsableID: "u-doble", PresupuestoAsignado: 60000, PresupuestoConsumido: 0, Avance: 0, Dependencias: []string{"e4"}},
}

var EntregablesSemilla = []domain.Entregable{
	{ID: "ent-1", ProyectoID: "p1", Etapa: domain.EtapaPlanteamiento, Nombre: "Acta de planteamiento", Version: 1, AprobadoPorID: str("u-sponsor"), AprobadoEn: str("2026-08-20T10:00:00Z"), CreadoPorID: "u-pm"},
	{ID: "ent-2", ProyectoID: "p1", Etapa: domain.EtapaIniciativa, Nombre: "Documento de iniciativa", Version: 1, AprobadoPorID: str("u-admin"), AprobadoEn: str("2026-08-28T10:00:00Z"), CreadoPorID: "u-pm"},
	{ID: "ent-3", ProyectoID: "p1", Etapa: domain.EtapaPoC, Nombre: "Plan de prueba de concepto", Version: 1, AprobadoPorID: str("u-sponsor"), AprobadoEn: str("2026-09-05T10:00:00Z"), CreadoPorID: "u-doble"},
	{ID: "ent-4", ProyectoID: "p1", Etapa: domain.EtapaValidacionPoC, Nombre: "Informe de validacion de PoC", Version: 1, AprobadoPorID: str("u-admin"), AprobadoEn: str("2026-09-12T10:00:00Z"), CreadoPorID: "u-pm"},
	{ID: "ent-5", ProyectoID: "p1", Etapa: domain.EtapaPlaneacion, Nombre: "Plan de proyecto", Version: 1, AprobadoPorID: str("u-sponsor"), AprobadoEn: str("2026-09-20T10:00:00Z"), CreadoPorID: "u-doble"},
	{ID: "ent-6", ProyectoID: "p1", Etapa: domain.EtapaContratos, Nombre: "Contrato o acta de compromisos", Version: 1, AprobadoPorID: str("u-admin"), AprobadoEn: str("2026-09-28T10:00:00Z"), CreadoPorID: "u-pm"},
	{ID: "ent-7", ProyectoID: "p1", Etapa: domain.EtapaEjecucion, Nombre: "Acta de avance de ejecucion", Version: 1, CreadoPorID: "u-doble"},
	{ID: "ent-8", ProyectoID: "p3", Etapa: domain.EtapaPlanteamiento, Nombre: "Acta de planteamiento", Version: 1, AprobadoPorID: str("u-admin"), AprobadoEn: str("2026-10-01T10:00:00Z"), CreadoPorID: "u-analista"},
	{ID: "ent-9", ProyectoID: "p3", Etapa: domain.EtapaIniciativa, Nombre: "Documento de iniciativa", Version: 1, AprobadoPorID: str("u-sponsor"), AprobadoEn: str("2026-10-10T10:00:00Z"), CreadoPorID: "u-analista"},
	{ID: "ent-10", ProyectoID: "p3", Etapa: domain.EtapaPoC, Nombre: "Plan de prueba de concepto", Version: 1, CreadoPorID: "u-analista"},
}

var BitacoraSemilla = []domain.BitacoraEntrada{
	{ID: "b1", ProyectoID: "p1", AutorID: "u-admin", Fecha: "2026-08-20T11:00:00Z", Tipo: domain.TipoTransicion, EtapaOrigen: etapa(domain.EtapaPlanteamiento), EtapaDestino: etapa(domain.EtapaIniciativa), Comentario: str("Acta aprobada por el sponsor.")},
	{ID: "b2", ProyectoID: "p1", AutorID: "u-admin", Fecha: "2026-08-28T11:00:00Z", Tipo: domain.TipoTransicion, EtapaOrigen: etapa(domain.EtapaIniciativa), EtapaDestino: etapa(domain.EtapaPoC)},
	{ID: "b3", ProyectoID: "p1", AutorID: "u-sponsor", Fecha: "2026-09-12T11:00:00Z", Tipo: domain.TipoTransicion, EtapaOrigen: etapa(domain.EtapaPoC), EtapaDestino: etapa(domain.EtapaValidacionPoC), Comentario: str("PoC cumple criterios.")},
	{ID: "b4", ProyectoID: "p1", AutorID: "u-admin", Fecha: "2026-09-28T11:30:00Z", Tipo: domain.TipoTransicion, EtapaOrigen: etapa(domain.EtapaContratos), EtapaDestino: etapa(domain.EtapaEjecucion), Comentario: str("Contrato firmado.")},
	{ID: "b5", ProyectoID: "p6", AutorID: "u-admin", Fecha: "2026-10-15T09:00:00Z", Tipo: domain.TipoCambio, Comentario: str("Proyecto cancelado por reorganizacion.")},
}
