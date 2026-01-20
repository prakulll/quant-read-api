package components

import (
	"time"

	"quant-read-api/services"
)

func GetOptionExpiries(
	underlying string,
	from *time.Time,
	to *time.Time,
) ([]string, error) {

	db := services.GetClickHouse()

	query := `
		SELECT DISTINCT expiry
		FROM options_moneyness
		WHERE underlying = ?
	`

	args := []any{underlying}

	if from != nil && to != nil {
		query += " AND ts BETWEEN ? AND ?"
		args = append(args, *from, *to)
	}

	query += " ORDER BY expiry"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expiries := make([]string, 0)

	for rows.Next() {
		var expiry time.Time
		if err := rows.Scan(&expiry); err != nil {
			return nil, err
		}
		expiries = append(expiries, expiry.Format("2006-01-02"))
	}

	return expiries, nil
}
