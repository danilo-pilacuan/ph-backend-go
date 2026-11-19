// Package domain: calculos de dominio (espejo de calculos.py, 4.3-4.5, 5.2,
// 5.4). Puros y sin framework; las rutas no los reimplementan.
package domain

import (
	"math"
	"strings"
	"time"
)

// DuracionDias: (fin - inicio) en dias + 1, como duracionDias de TS.
// Error VALIDACION si las fechas no son ISO o fin < inicio.
func DuracionDias(fechaInicio, fechaFin string) (int, error) {
	inicio, err := time.Parse("2006-01-02", fechaInicio)
	if err != nil {
		return 0, ErrorValidacion("Las fechas de la etapa deben ser fechas validas.")
	}
	fin, err := time.Parse("2006-01-02", fechaFin)
	if err != nil {
		return 0, ErrorValidacion("Las fechas de la etapa deben ser fechas validas.")
	}
	if fin.Before(inicio) {
		return 0, ErrorValidacion("La fecha de fin no puede ser anterior a la fecha de inicio.")
	}
	dias := int(fin.Sub(inicio).Hours()/24) + 1
	if dias < 1 {
		dias = 1
	}
	return dias, nil
}

// ValidarEtapa: fechas coherentes, avance 0-100 y montos finitos no negativos.
func ValidarEtapa(fechaInicio, fechaFin string, avance, presupuestoAsignado, presupuestoConsumido float64) error {
	if _, err := DuracionDias(fechaInicio, fechaFin); err != nil {
		return err
	}
	if !esMontoValido(avance) || avance > 100 {
		return ErrorValidacion("El avance de la etapa debe estar entre 0 y 100.")
	}
	if !esMontoValido(presupuestoAsignado) {
		return ErrorValidacion("El presupuesto asignado no puede ser negativo.")
	}
	if !esMontoValido(presupuestoConsumido) {
		return ErrorValidacion("El presupuesto consumido no puede ser negativo.")
	}
	return nil
}

func esMontoValido(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

func Redondear2(v float64) float64 {
	return math.Round(v*100) / 100
}

// AvancePonderado: promedio ponderado por duracion (4.4); 0 si no hay etapas.
func AvancePonderado(etapas []EtapaTrabajo) (float64, error) {
	if len(etapas) == 0 {
		return 0, nil
	}
	sumaPonderada := 0.0
	sumaDuraciones := 0
	for _, e := range etapas {
		if err := ValidarEtapa(e.FechaInicio, e.FechaFin, e.Avance, e.PresupuestoAsignado, e.PresupuestoConsumido); err != nil {
			return 0, err
		}
		dias, err := DuracionDias(e.FechaInicio, e.FechaFin)
		if err != nil {
			return 0, err
		}
		sumaPonderada += e.Avance * float64(dias)
		sumaDuraciones += dias
	}
	return Redondear2(sumaPonderada / float64(sumaDuraciones)), nil
}

// PresupuestoTotal: suma de asignados (5.2).
func PresupuestoTotal(etapas []EtapaTrabajo) (float64, error) {
	total := 0.0
	for _, e := range etapas {
		if err := ValidarEtapa(e.FechaInicio, e.FechaFin, e.Avance, e.PresupuestoAsignado, e.PresupuestoConsumido); err != nil {
			return 0, err
		}
		total += e.PresupuestoAsignado
	}
	return Redondear2(total), nil
}

// ConsumoTotal: suma de consumidos (5.2).
func ConsumoTotal(etapas []EtapaTrabajo) (float64, error) {
	total := 0.0
	for _, e := range etapas {
		if err := ValidarEtapa(e.FechaInicio, e.FechaFin, e.Avance, e.PresupuestoAsignado, e.PresupuestoConsumido); err != nil {
			return 0, err
		}
		total += e.PresupuestoConsumido
	}
	return Redondear2(total), nil
}

// DesviacionPresupuesto: consumido - asignado (5.4); > 0 => excedido.
func DesviacionPresupuesto(etapa EtapaTrabajo) (float64, error) {
	if err := ValidarEtapa(etapa.FechaInicio, etapa.FechaFin, etapa.Avance, etapa.PresupuestoAsignado, etapa.PresupuestoConsumido); err != nil {
		return 0, err
	}
	return Redondear2(etapa.PresupuestoConsumido - etapa.PresupuestoAsignado), nil
}

func PresupuestoExcedido(etapa EtapaTrabajo) bool {
	d, err := DesviacionPresupuesto(etapa)
	return err == nil && d > 0
}

// EsCorreoValido: regex espejo de _CORREO_RE de servicio.py.
func EsCorreoValido(correo string) bool {
	arroba := strings.Index(correo, "@")
	if arroba <= 0 || arroba == len(correo)-1 {
		return false
	}
	dominio := correo[arroba+1:]
	punto := strings.LastIndex(dominio, ".")
	return punto > 0 && punto < len(dominio)-1 && !strings.ContainsAny(correo, " \t\n")
}
