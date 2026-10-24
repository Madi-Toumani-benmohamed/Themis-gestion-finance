// Package money implements exact euro cent amounts. Float64 is accepted only
// at the legacy model boundary, never for arithmetic or silent rounding.
package money

import (
	"fmt"
	"strconv"
	"strings"
)

type Cents int64

// Parse rejects sub-cent precision and amounts outside PostgreSQL NUMERIC(12,2).
func Parse(s string) (Cents, error) {
	negative := strings.HasPrefix(s, "-")
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, fmt.Errorf("montant invalide")
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return 0, fmt.Errorf("montant invalide")
		}
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, fmt.Errorf("précision monétaire invalide")
		}
		fraction = (parts[1] + "0")[:2]
	}
	units, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || units > 9999999999 {
		return 0, fmt.Errorf("montant hors limites")
	}
	cents, _ := strconv.ParseInt(fraction, 10, 64)
	value := Cents(units*100 + cents)
	if negative {
		value = -value
	}
	return value, nil
}
func FromLegacy(v float64) (Cents, error) { return Parse(strconv.FormatFloat(v, 'f', -1, 64)) }
func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign = "-"
		c = -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}
