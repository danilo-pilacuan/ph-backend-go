// Package domain: grafo de transiciones del ciclo de vida (espejo de
// ciclo_vida.py). Avances: cadena lineal Planteamiento -> ... -> Cierre.
// Retrocesos permitidos (3.3): solo PoC -> Iniciativa y ValidacionPoC -> PoC.
package domain

var Retrocesos = map[EtapaCicloVida]EtapaCicloVida{
	EtapaPoC:           EtapaIniciativa,
	EtapaValidacionPoC: EtapaPoC,
}

func SiguienteEtapa(etapa EtapaCicloVida) (EtapaCicloVida, bool) {
	i := IndiceEtapa[etapa]
	if i < len(EtapasCicloVida)-1 {
		return EtapasCicloVida[i+1], true
	}
	return "", false
}

func EtapaRetrocesoPermitido(etapa EtapaCicloVida) (EtapaCicloVida, bool) {
	r, ok := Retrocesos[etapa]
	return r, ok
}

func EsTransicionValida(origen, destino EtapaCicloVida) bool {
	if sig, ok := SiguienteEtapa(origen); ok && sig == destino {
		return true
	}
	if retro, ok := EtapaRetrocesoPermitido(origen); ok {
		return retro == destino
	}
	return false
}

func ValidarTransicion(origen, destino EtapaCicloVida) error {
	if !EsTransicionValida(origen, destino) {
		return ErrorTransicionInvalida(origen, destino)
	}
	return nil
}
