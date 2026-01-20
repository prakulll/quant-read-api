package components

import (
	"time"

	"quant-read-api/models"
	"quant-read-api/services"
)

func GetOptionContractsByPremium(
	underlying string,
	optionType string, // CE | PE | BOTH
	targetPremium float64,
	tolerance float64,
	expiryMode string,
	from time.Time,
	to time.Time,
	limit int,
) ([]models.OptionSnapshot, error) {

	db := services.GetClickHouse()

	lower := targetPremium - tolerance
	upper := targetPremium + tolerance

	// option_type filter
	optionFilter := "option_type = ?"
	if optionType == "BOTH" {
		optionFilter = "option_type IN ('CE','PE')"
	}

	// expiry filter
	var expirySQL string
	var expiryArgs []any

	if expiryMode != "all" {
		if optionType == "BOTH" {
			expirySQL = `
				AND expiry = (
					SELECT min(expiry)
					FROM options_moneyness
					WHERE underlying = ?
					  AND option_type IN ('CE','PE')
					  AND expiry >= toDate(?)
				)
			`
			expiryArgs = append(expiryArgs, underlying, from.Format("2006-01-02"))
		} else {
			expirySQL = `
				AND expiry = (
					SELECT min(expiry)
					FROM options_moneyness
					WHERE underlying = ?
					  AND option_type = ?
					  AND expiry >= toDate(?)
				)
			`
			expiryArgs = append(expiryArgs, underlying, optionType, from.Format("2006-01-02"))
		}
	}

	query := `
		SELECT
			ts,
			underlying,
			expiry,
			strike,
			option_type,
			ltp,
			spot_price,
			atm_strike,
			moneyness,
			moneyness_lvl,
			days_to_expiry
		FROM options_moneyness
		WHERE underlying = ?
		  AND ` + optionFilter + `
		  AND ts BETWEEN ? AND ?
		  AND ltp BETWEEN ? AND ?
		  ` + expirySQL + `
		ORDER BY abs(moneyness_lvl) ASC
		LIMIT ?
	`

	args := []any{underlying}

	if optionType != "BOTH" {
		args = append(args, optionType)
	}

	args = append(args, from, to, lower, upper)
	args = append(args, expiryArgs...)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.OptionSnapshot, 0)

	for rows.Next() {
		var r models.OptionSnapshot
		if err := rows.Scan(
			&r.Ts,
			&r.Underlying,
			&r.Expiry,
			&r.Strike,
			&r.OptionType,
			&r.Ltp,
			&r.SpotPrice,
			&r.AtmStrike,
			&r.Moneyness,
			&r.MoneynessLvl,
			&r.DaysToExpiry,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}

	return out, nil
}
