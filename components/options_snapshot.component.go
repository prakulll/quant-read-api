package components

import (
	"sync"
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

// ==================================================
// V2 — Concurrent snapshot by expiry
// ==================================================
func GetOptionSnapshotsConcurrent(
	underlying string,
	optionType string,
	moneyness string,
	moneynessMode string,
	moneynessLvl int,
	expiryMode string,
	from time.Time,
	to time.Time,
) (map[string]models.OptionSnapshotColumnar, []string, error) {

	expiries, err := GetOptionExpiries(underlying, &from, &to)
	if err != nil {
		return nil, nil, err
	}

	if len(expiries) == 0 {
		return map[string]models.OptionSnapshotColumnar{}, expiries, nil
	}

	if expiryMode == "nearest" && len(expiries) > 1 {
		expiries = expiries[:1]
	}

	// HARD CAP — mandatory for scale
	const MAX_EXPIRIES = 6
	if len(expiries) > MAX_EXPIRIES {
		expiries = expiries[:MAX_EXPIRIES]
	}

	out := make(map[string]models.OptionSnapshotColumnar)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, exp := range expiries {
		expiry := exp
		wg.Add(1)

		go func() {
			defer wg.Done()

			rows, err := GetOptionSnapshots(
				underlying,
				optionType,
				moneyness,
				moneynessMode,
				moneynessLvl,
				expiry, // ← NOW CORRECT
				from,
				to,
			)
			if err != nil {
				return
			}

			snap := models.OptionSnapshotColumnar{
				Ts:           make([]time.Time, 0, len(rows)),
				Underlying:   make([]string, 0, len(rows)),
				Expiry:       make([]time.Time, 0, len(rows)),
				Strike:       make([]uint32, 0, len(rows)),
				OptionType:   make([]string, 0, len(rows)),
				Ltp:          make([]float64, 0, len(rows)),
				SpotPrice:    make([]float64, 0, len(rows)),
				AtmStrike:    make([]uint32, 0, len(rows)),
				Moneyness:    make([]string, 0, len(rows)),
				MoneynessLvl: make([]int16, 0, len(rows)),
				DaysToExpiry: make([]int16, 0, len(rows)),
			}

			for _, r := range rows {
				snap.Ts = append(snap.Ts, r.Ts)
				snap.Underlying = append(snap.Underlying, r.Underlying)
				snap.Expiry = append(snap.Expiry, r.Expiry)
				snap.Strike = append(snap.Strike, r.Strike)
				snap.OptionType = append(snap.OptionType, r.OptionType)
				snap.Ltp = append(snap.Ltp, r.Ltp)
				snap.SpotPrice = append(snap.SpotPrice, r.SpotPrice)
				snap.AtmStrike = append(snap.AtmStrike, r.AtmStrike)
				snap.Moneyness = append(snap.Moneyness, r.Moneyness)
				snap.MoneynessLvl = append(snap.MoneynessLvl, r.MoneynessLvl)
				snap.DaysToExpiry = append(snap.DaysToExpiry, r.DaysToExpiry)
			}

			mu.Lock()
			out[expiry] = snap
			mu.Unlock()
		}()
	}

	wg.Wait()
	return out, expiries, nil
}
