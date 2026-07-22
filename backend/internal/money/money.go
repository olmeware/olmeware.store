// Package money formats integer minor-unit amounts for display.
package money

import "strconv"

// FormatMXN renders a minor-unit amount as a Mexican-peso string matching the
// storefront's formatter, e.g. 44900 -> "$449 MXN", 129900 -> "$1,299 MXN".
func FormatMXN(minor int64) string {
	pesos := minor / 100
	neg := pesos < 0
	if neg {
		pesos = -pesos
	}
	digits := strconv.FormatInt(pesos, 10)

	var grouped []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped = append(grouped, ',')
		}
		grouped = append(grouped, d)
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return sign + "$" + string(grouped) + " MXN"
}
