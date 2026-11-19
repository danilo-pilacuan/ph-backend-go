// Command golden-seed: vuelca la semilla Go en JSON camelCase para comparar
// byte a byte con el volcado de app/semilla.py (golden test de paridad).
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/danilo-pilacuan/projecthub-backend-go/internal/semilla"
)

type envoltura struct {
	Usuarios    []any `json:"usuarios"`
	Proyectos   []any `json:"proyectos"`
	Etapas      []any `json:"etapas"`
	Entregables []any `json:"entregables"`
	Bitacora    []any `json:"bitacora"`
}

func main() {
	e := envoltura{}
	for _, u := range semilla.UsuariosSemilla {
		e.Usuarios = append(e.Usuarios, u)
	}
	for _, p := range semilla.ProyectosSemilla {
		e.Proyectos = append(e.Proyectos, p)
	}
	for _, x := range semilla.EtapasSemilla {
		e.Etapas = append(e.Etapas, x)
	}
	for _, x := range semilla.EntregablesSemilla {
		e.Entregables = append(e.Entregables, x)
	}
	for _, x := range semilla.BitacoraSemilla {
		e.Bitacora = append(e.Bitacora, x)
	}
	salida, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(salida)
	os.Stdout.Write([]byte("\n"))
}
