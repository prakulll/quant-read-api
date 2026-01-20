package components

import (
	"time"

	"quant-read-api/models"
	"quant-read-api/services"
)

func GetOptionSnapshots(
	underlying string,
	optionType string,
	moneyness string,
	moneynessMode string,
	moneynessLvl int,
	expiryMode string,
	from time.Time,
	to time.Time,
) ([]models.OptionSnapshot, error) {

	db := services.GetClickHouse()

	var query string
	var args []any

	// ----- moneyness condition -----
	var moneynessSQL string

	if moneyness == "ATM" {
		moneynessSQL = "AND moneyness = 'ATM' AND moneyness_lvl = 0"
	} else if moneynessMode == "exact" {
		moneynessSQL = "AND moneyness = ? AND moneyness_lvl = ?"
		args = append(args, moneyness, moneynessLvl)
	} else if moneyness == "ALL" {
		moneynessSQL = "AND moneyness_lvl BETWEEN -? AND ?"
		args = append(args, moneynessLvl, moneynessLvl)
	} else {
		moneynessSQL = "AND moneyness = ? AND moneyness_lvl BETWEEN 1 AND ?"
		args = append(args, moneyness, moneynessLvl)
	}

	// ----- expiry condition -----
	var expirySQL string
	if expiryMode == "all" {
		expirySQL = ""
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
		args = append(args, underlying, optionType, from.Format("2006-01-02"))
	}

	query = `
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
		  AND option_type = ?
		  AND ts BETWEEN ? AND ?
		  ` + moneynessSQL + `
		  ` + expirySQL + `
		ORDER BY ts, strike, days_to_expiry
	`

	finalArgs := []any{
		underlying,
		optionType,
		from,
		to,
	}
	finalArgs = append(finalArgs, args...)

	rows, err := db.Query(query, finalArgs...)
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
