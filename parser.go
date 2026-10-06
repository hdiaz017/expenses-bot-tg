package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseExpense(text string) (string, float64, error) {
	parts := strings.Fields(text)

	if len(parts) < 2 {
		return "", 0, fmt.Errorf(
			"formato inválido. Ejemplo: Torta 85",
		)
	}

	amountText := parts[len(parts)-1]

	amount, err := strconv.ParseFloat(amountText, 64)
	if err != nil {
		return "", 0, fmt.Errorf(
			"no pude interpretar el monto: %s",
			amountText,
		)
	}

	concept := strings.Join(parts[:len(parts)-1], " ")

	if concept == "" {
		return "", 0, fmt.Errorf("falta el concepto")
	}

	return concept, amount, nil
}
