// Rutas REST (espejo de app/api/rutas.py): mismos metodos, paths y codigos de
// estado. Solo /api/salud y /api/auth/login son publicos; el resto exige
// actor. Los codigos de ErrorDominio se mapean a HTTP en mapearErrorDominio.
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/auth"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/domain"
	"github.com/danilo-pilacuan/projecthub-backend-go/internal/servicio"
)

func (s *Servidor) registrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/salud", s.salud)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/auth/me", s.conActor(s.usuarioActual))
	mux.HandleFunc("PUT /api/auth/contrasena", s.conActor(s.cambiarContrasena))
	mux.HandleFunc("GET /api/usuarios", s.conActor(s.listarUsuarios))
	mux.HandleFunc("POST /api/usuarios", s.conActor(s.crearUsuario))
	mux.HandleFunc("PATCH /api/usuarios/{usuarioID}", s.conActor(s.editarUsuario))
	mux.HandleFunc("GET /api/proyectos", s.conActor(s.listarProyectos))
	mux.HandleFunc("POST /api/proyectos", s.conActor(s.crearProyecto))
	mux.HandleFunc("GET /api/proyectos/resumenes", s.conActor(s.listarResumenes))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}", s.conActor(s.obtenerProyecto))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}/resumen", s.conActor(s.obtenerResumen))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}/gate", s.conActor(s.obtenerGate))
	mux.HandleFunc("PATCH /api/proyectos/{proyectoID}", s.conActor(s.editarProyecto))
	mux.HandleFunc("POST /api/proyectos/{proyectoID}/transicionar", s.conActor(s.transicionar))
	mux.HandleFunc("POST /api/proyectos/{proyectoID}/cancelar", s.conActor(s.cancelarProyecto))
	mux.HandleFunc("POST /api/proyectos/{proyectoID}/cerrar", s.conActor(s.cerrarProyecto))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}/etapas", s.conActor(s.listarEtapas))
	mux.HandleFunc("POST /api/proyectos/{proyectoID}/etapas", s.conActor(s.crearEtapa))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}/entregables", s.conActor(s.listarEntregables))
	mux.HandleFunc("POST /api/proyectos/{proyectoID}/entregables", s.conActor(s.crearEntregable))
	mux.HandleFunc("GET /api/proyectos/{proyectoID}/bitacora", s.conActor(s.listarBitacora))
	mux.HandleFunc("PATCH /api/etapas/{etapaID}", s.conActor(s.editarEtapa))
	mux.HandleFunc("DELETE /api/etapas/{etapaID}", s.conActor(s.eliminarEtapa))
	mux.HandleFunc("POST /api/etapas/{etapaID}/avance", s.conActor(s.registrarAvance))
	mux.HandleFunc("POST /api/etapas/{etapaID}/consumo", s.conActor(s.registrarConsumo))
	mux.HandleFunc("GET /api/etapas", s.conActor(s.listarTodasLasEtapas))
	mux.HandleFunc("POST /api/entregables/{entregableID}/aprobar", s.conActor(s.aprobarEntregable))
	mux.HandleFunc("GET /api/entregables", s.conActor(s.listarTodosLosEntregables))
	mux.HandleFunc("GET /api/bitacora", s.conActor(s.listarTodaLaBitacora))
	mux.HandleFunc("GET /api/sincronizacion", s.conActor(s.leerEstado))
	mux.HandleFunc("PUT /api/sincronizacion", s.conActor(s.escribirEstado))
}

// conActor: resuelve el actor y delega; errores HTTP y de dominio mapeados.
func (s *Servidor) conActor(proximo func(w http.ResponseWriter, r *http.Request, actor domain.Usuario)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.obtenerActor(w, r)
		if err != nil {
			s.responderError(w, err)
			return
		}
		proximo(w, r, actor)
	}
}

func (s *Servidor) responderError(w http.ResponseWriter, err error) {
	if he, ok := err.(*errorHTTPTipo); ok {
		escribirError(w, he.estado, he.codigo, he.mensaje)
		return
	}
	mapearErrorDominio(w, err)
}

// ---------- Salud ----------

func (s *Servidor) salud(w http.ResponseWriter, r *http.Request) {
	codificar(w, http.StatusOK, map[string]string{"estado": "ok", "servicio": "projecthub-backend"})
}

// ---------- Autenticacion ----------

func (s *Servidor) login(w http.ResponseWriter, r *http.Request) {
	var cuerpo CuerpoLogin
	if !decodificar(w, r, &cuerpo) {
		return
	}
	clave := strings.ToLower(strings.TrimSpace(cuerpo.Correo)) + "|" + ipCliente(r)
	if s.Limitador.bloqueado(clave, tiempoAhora()) {
		escribirError(w, 429, "LIMITE_INTENTOS", "Demasiados intentos fallidos; reintente mas tarde.")
		return
	}
	var usuario *domain.Usuario
	for _, u := range s.Servicio.Usuarios() {
		if strings.EqualFold(u.Correo, cuerpo.Correo) {
			u := u
			usuario = &u
			break
		}
	}
	if usuario == nil || !s.Credenciales.Autenticar(usuario.Correo, cuerpo.Contrasena) {
		s.Limitador.registrarFallo(clave, tiempoAhora())
		escribirError(w, 401, "CREDENCIALES_INVALIDAS", "Correo o contrasena incorrectos.")
		return
	}
	if !usuario.Activo {
		escribirError(w, 403, "USUARIO_INACTIVO", "Un usuario inactivo no puede iniciar sesion.")
		return
	}
	codificar(w, http.StatusOK, map[string]any{
		"token":   auth.CrearToken(usuario.ID, s.Secreto, nil),
		"tipo":    "Bearer",
		"usuario": usuario,
	})
}

func (s *Servidor) usuarioActual(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	codificar(w, http.StatusOK, actor)
}

func (s *Servidor) cambiarContrasena(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoCambioContrasena
	if !decodificar(w, r, &cuerpo) {
		return
	}
	cambiada, err := s.Credenciales.Cambiar(actor.Correo, cuerpo.ContrasenaActual, cuerpo.ContrasenaNueva)
	if err != nil {
		s.responderError(w, err)
		return
	}
	if !cambiada {
		escribirError(w, 401, "CONTRASENA_ACTUAL_INVALIDA", "La contrasena actual no es correcta.")
		return
	}
	codificar(w, http.StatusOK, map[string]bool{"cambiada": true})
}

// ---------- Usuarios ----------

func (s *Servidor) listarUsuarios(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	fuente := s.Servicio.Usuarios()
	if r.URL.Query().Get("solo_activos") == "true" || r.URL.Query().Get("soloActivos") == "true" {
		fuente = s.Servicio.UsuariosActivos()
	}
	if fuente == nil {
		fuente = []domain.Usuario{}
	}
	codificar(w, http.StatusOK, fuente)
}

func (s *Servidor) crearUsuario(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoUsuario
	if !decodificar(w, r, &cuerpo) {
		return
	}
	roles := cuerpo.Roles
	if roles == nil {
		roles = []domain.Rol{}
	}
	activo := true
	if cuerpo.Activo != nil {
		activo = *cuerpo.Activo
	}
	creado, err := s.Servicio.CrearUsuario(servicio.DatosUsuario{
		Nombre: &cuerpo.Nombre, Correo: &cuerpo.Correo, Roles: &roles, Activo: &activo,
	}, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	// Credencial demo inicial (igual que rutas.py).
	s.Credenciales.Registrar(creado.Correo, auth.ClaveDemo())
	codificar(w, http.StatusCreated, creado)
}

func (s *Servidor) editarUsuario(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoEdicionUsuario
	if !decodificar(w, r, &cuerpo) {
		return
	}
	datos := servicio.DatosUsuario{Nombre: cuerpo.Nombre, Correo: cuerpo.Correo, Roles: cuerpo.Roles, Activo: cuerpo.Activo}
	if _, err := s.Servicio.EditarUsuario(r.PathValue("usuarioID"), datos, actor); err != nil {
		s.responderError(w, err)
		return
	}
	actualizado := s.Servicio.UsuarioPorID(r.PathValue("usuarioID"))
	if actualizado == nil {
		s.responderError(w, domain.ErrorEntidadNoEncontrada("Usuario", r.PathValue("usuarioID")))
		return
	}
	codificar(w, http.StatusOK, actualizado)
}

// ---------- Proyectos ----------

func (s *Servidor) listarProyectos(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.Proyectos()
	if lista == nil {
		lista = []domain.Proyecto{}
	}
	codificar(w, http.StatusOK, lista)
}

func (s *Servidor) crearProyecto(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoProyecto
	if !decodificar(w, r, &cuerpo) {
		return
	}
	datos := servicio.DatosProyecto{Nombre: &cuerpo.Nombre, Descripcion: &cuerpo.Descripcion, PatrocinadorID: cuerpo.PatrocinadorID, FechaObjetivo: cuerpo.FechaObjetivo}
	if cuerpo.PresupuestoEstimado != nil {
		datos.PresupuestoEstimado = cuerpo.PresupuestoEstimado
	}
	creado, err := s.Servicio.CrearProyecto(datos, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusCreated, creado)
}

func resumenJSON(rs domain.ResumenProyecto) map[string]any {
	etapas := rs.Etapas
	if etapas == nil {
		etapas = []domain.EtapaTrabajo{}
	}
	return map[string]any{
		"proyecto":          rs.Proyecto,
		"avance":            rs.Avance,
		"presupuestoTotal":  rs.PresupuestoTotal,
		"consumoTotal":      rs.ConsumoTotal,
		"excedido":          rs.Excedido,
		"etapas":            etapas,
	}
}

func (s *Servidor) listarResumenes(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	salida := []map[string]any{}
	for _, rs := range s.Servicio.Resumenes() {
		salida = append(salida, resumenJSON(rs))
	}
	codificar(w, http.StatusOK, salida)
}

func (s *Servidor) obtenerProyecto(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	proyecto := s.Servicio.ProyectoPorID(r.PathValue("proyectoID"))
	if proyecto == nil {
		s.responderError(w, domain.ErrorEntidadNoEncontrada("Proyecto", r.PathValue("proyectoID")))
		return
	}
	codificar(w, http.StatusOK, proyecto)
}

func (s *Servidor) obtenerResumen(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	rs, err := s.Servicio.Resumen(r.PathValue("proyectoID"))
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, resumenJSON(rs))
}

func (s *Servidor) obtenerGate(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	gate, err := s.Servicio.GateActual(r.PathValue("proyectoID"))
	if err != nil {
		s.responderError(w, err)
		return
	}
	entregables := gate.Entregables
	if entregables == nil {
		entregables = []domain.Entregable{}
	}
	codificar(w, http.StatusOK, map[string]any{
		"etapa":                gate.Etapa,
		"abierto":              gate.Abierto,
		"entregables":          entregables,
		"faltantes":            gate.Faltantes,
		"pendientesAprobacion": gate.PendientesAprobacion,
	})
}

func (s *Servidor) editarProyecto(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoEdicionProyecto
	if !decodificar(w, r, &cuerpo) {
		return
	}
	datos := servicio.DatosProyecto{Nombre: cuerpo.Nombre, Descripcion: cuerpo.Descripcion, FechaObjetivo: cuerpo.FechaObjetivo}
	if _, err := s.Servicio.EditarProyecto(r.PathValue("proyectoID"), datos, actor); err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, s.Servicio.ProyectoPorID(r.PathValue("proyectoID")))
}

func (s *Servidor) transicionar(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoTransicion
	if !decodificar(w, r, &cuerpo) {
		return
	}
	if _, err := s.Servicio.Transicionar(r.PathValue("proyectoID"), cuerpo.Destino, actor, cuerpo.Comentario); err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, s.Servicio.ProyectoPorID(r.PathValue("proyectoID")))
}

func (s *Servidor) cancelarProyecto(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo *CuerpoComentario
	_ = decodificarOpcional(w, r, &cuerpo)
	comentario := (*string)(nil)
	if cuerpo != nil {
		comentario = cuerpo.Comentario
	}
	if _, err := s.Servicio.CancelarProyecto(r.PathValue("proyectoID"), actor, comentario); err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, s.Servicio.ProyectoPorID(r.PathValue("proyectoID")))
}

func (s *Servidor) cerrarProyecto(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo *CuerpoComentario
	_ = decodificarOpcional(w, r, &cuerpo)
	comentario := (*string)(nil)
	if cuerpo != nil {
		comentario = cuerpo.Comentario
	}
	if _, err := s.Servicio.CerrarProyecto(r.PathValue("proyectoID"), actor, comentario); err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, s.Servicio.ProyectoPorID(r.PathValue("proyectoID")))
}

// ---------- Etapas de trabajo ----------

func (s *Servidor) listarEtapas(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.EtapasDeProyecto(r.PathValue("proyectoID"))
	if lista == nil {
		lista = []domain.EtapaTrabajo{}
	}
	codificar(w, http.StatusOK, lista)
}

func (s *Servidor) crearEtapa(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoEtapa
	if !decodificar(w, r, &cuerpo) {
		return
	}
	datos := servicio.DatosEtapa{
		Nombre: &cuerpo.Nombre, FechaInicio: &cuerpo.FechaInicio, FechaFin: &cuerpo.FechaFin,
		ResponsableID: &cuerpo.ResponsableID, PresupuestoAsignado: cuerpo.PresupuestoAsignado,
		Avance: cuerpo.Avance, Dependencias: cuerpo.Dependencias,
	}
	creada, err := s.Servicio.CrearEtapa(r.PathValue("proyectoID"), datos, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusCreated, creada)
}

func (s *Servidor) editarEtapa(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoEdicionEtapa
	if !decodificar(w, r, &cuerpo) {
		return
	}
	datos := servicio.DatosEtapa{
		Nombre: cuerpo.Nombre, FechaInicio: cuerpo.FechaInicio, FechaFin: cuerpo.FechaFin,
		ResponsableID: cuerpo.ResponsableID, PresupuestoAsignado: cuerpo.PresupuestoAsignado,
		Avance: cuerpo.Avance, Dependencias: cuerpo.Dependencias,
	}
	actualizada, err := s.Servicio.EditarEtapa(r.PathValue("etapaID"), datos, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, actualizada)
}

func (s *Servidor) eliminarEtapa(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	if err := s.Servicio.EliminarEtapa(r.PathValue("etapaID"), actor); err != nil {
		s.responderError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servidor) registrarAvance(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoAvance
	if !decodificar(w, r, &cuerpo) {
		return
	}
	actualizada, err := s.Servicio.RegistrarAvance(r.PathValue("etapaID"), cuerpo.Avance, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, actualizada)
}

func (s *Servidor) registrarConsumo(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoConsumo
	if !decodificar(w, r, &cuerpo) {
		return
	}
	actualizada, err := s.Servicio.RegistrarConsumo(r.PathValue("etapaID"), cuerpo.Consumo, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, actualizada)
}

// ---------- Entregables ----------

func (s *Servidor) listarEntregables(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.EntregablesDeProyecto(r.PathValue("proyectoID"))
	if lista == nil {
		lista = []domain.Entregable{}
	}
	codificar(w, http.StatusOK, lista)
}

func (s *Servidor) crearEntregable(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var cuerpo CuerpoEntregable
	if !decodificar(w, r, &cuerpo) {
		return
	}
	version := 1
	if cuerpo.Version != nil {
		version = *cuerpo.Version
	}
	creado, err := s.Servicio.CrearEntregable(r.PathValue("proyectoID"), cuerpo.Etapa, cuerpo.Nombre, version, actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusCreated, creado)
}

func (s *Servidor) aprobarEntregable(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	aprobado, err := s.Servicio.AprobarEntregable(r.PathValue("entregableID"), actor)
	if err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, aprobado)
}

// ---------- Bitacora ----------

func (s *Servidor) listarBitacora(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.BitacoraDeProyecto(r.PathValue("proyectoID"))
	if lista == nil {
		lista = []domain.BitacoraEntrada{}
	}
	codificar(w, http.StatusOK, lista)
}

// ---------- Colecciones planas ----------

func (s *Servidor) listarTodasLasEtapas(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.Etapas()
	if lista == nil {
		lista = []domain.EtapaTrabajo{}
	}
	codificar(w, http.StatusOK, lista)
}

func (s *Servidor) listarTodosLosEntregables(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.Entregables()
	if lista == nil {
		lista = []domain.Entregable{}
	}
	codificar(w, http.StatusOK, lista)
}

func (s *Servidor) listarTodaLaBitacora(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	lista := s.Servicio.Bitacora()
	if lista == nil {
		lista = []domain.BitacoraEntrada{}
	}
	codificar(w, http.StatusOK, lista)
}

// ---------- Sincronizacion ----------

func (s *Servidor) leerEstado(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	codificar(w, http.StatusOK, s.Servicio.EstadoCompleto())
}

func (s *Servidor) escribirEstado(w http.ResponseWriter, r *http.Request, actor domain.Usuario) {
	var entrante servicio.EstadoCompleto
	if !decodificar(w, r, &entrante) {
		return
	}
	if err := s.Servicio.EscribirEstadoCompleto(entrante, actor); err != nil {
		s.responderError(w, err)
		return
	}
	codificar(w, http.StatusOK, s.Servicio.EstadoCompleto())
}

// decodificarOpcional: cuerpo JSON opcional (cancelar/cerrar admiten vacio).
func decodificarOpcional[T any](w http.ResponseWriter, r *http.Request, dst **T) bool {
	if r.ContentLength == 0 {
		return true
	}
	var cuerpo T
	if err := jsonNewDecoder(r.Body).Decode(&cuerpo); err != nil {
		escribirError(w, http.StatusUnprocessableEntity, "VALIDACION", "El cuerpo de la peticion no es JSON valido.")
		return false
	}
	*dst = &cuerpo
	return true
}

func jsonNewDecoder(r interface{ Read([]byte) (int, error) }) *jsonDecoder {
	return &jsonDecoder{d: json.NewDecoder(r)}
}

type jsonDecoder struct{ d *json.Decoder }

func (j *jsonDecoder) Decode(v any) error { return j.d.Decode(v) }

func ipCliente(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	if host == "" {
		return "desconocida"
	}
	return host
}

func tiempoAhora() time.Time { return time.Now() }
