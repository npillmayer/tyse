// percent implements a simple and straightforward type for percentage values
package percent

import (
	"math"
	"strconv"
	"strings"
)

// Percent is a simple and straightforward type for percentage values
type Percent uint8

func FromInt(n int) Percent {
	switch {
	case n <= 0:
		return Percent(0)
	case n >= 100:
		return Percent(100)
	}
	return Percent(n)
}

func FromFloat(f float64) Percent {
	switch {
	case f <= 0 || math.IsNaN(f) || math.IsInf(f, -1):
		return Percent(0)
	case f >= 100 || math.IsInf(f, 1):
		return Percent(100)
	}
	return Percent(math.Round(f))
}

func FromString(s string) (Percent, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	n, err := strconv.Atoi(s)
	return Percent(n), err
}

func (p Percent) String() string {
	return strconv.Itoa(int(p)) + "%"
}
