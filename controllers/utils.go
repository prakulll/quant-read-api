package controllers

import (
	"fmt"
	"strconv"
)

func parseTF(tf string) (int64, error) {
	if len(tf) < 2 {
		return 0, fmt.Errorf("invalid tf")
	}

	unit := tf[len(tf)-1]
	value, err := strconv.ParseInt(tf[:len(tf)-1], 10, 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 's':
		return value, nil
	case 'm':
		return value * 60, nil
	default:
		return 0, fmt.Errorf("invalid tf unit")
	}
}
