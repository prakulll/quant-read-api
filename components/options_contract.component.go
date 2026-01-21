package components

import (
	"time"

	"quant-read-api/models"
	"quant-read-api/services"
)

func GetOptionContract(
	underlying string,
	expiry time.Time,
	strike uint32,
	optionType string,
	from time.Time,
	to time.Time,
	tfSeconds *int64,
	offsetSeconds int64,
) (any, error) {

	db := services.GetClickHouse()
	loc, _ := time.LoadLocation("Asia/Kolkata")

	// =========================
	// RAW PATH (MULTI-DAY SAFE)
	// =========================
	if tfSeconds == nil {
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
			  AND expiry = toDate(?)
			  AND strike = ?
			  AND option_type = ?
			  AND ts >= ?
			  AND ts < ?
			ORDER BY ts
		`

		rows, err := db.Query(
			query,
			underlying,
			expiry,
			strike,
			optionType,
			from,
			to,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		out := []models.OptionContractRow{}

		for rows.Next() {
			var r models.OptionContractRow
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

	// =====================================
	// RESAMPLED PATH (MULTI-DAY, OFFSET OK)
	// =====================================

	type candle struct {
		Ts    time.Time
		Open  float64
		High  float64
		Low   float64
		Close float64
	}

	all := []candle{}

	day := time.Date(
		from.In(loc).Year(),
		from.In(loc).Month(),
		from.In(loc).Day(),
		0, 0, 0, 0, loc,
	)

	lastDay := time.Date(
		to.In(loc).Year(),
		to.In(loc).Month(),
		to.In(loc).Day(),
		0, 0, 0, 0, loc,
	)

	for !day.After(lastDay) {

		sessionStart := time.Date(
			day.Year(), day.Month(), day.Day(),
			9, 15, 0, 0, loc,
		)

		sessionEnd := time.Date(
			day.Year(), day.Month(), day.Day(),
			15, 30, 0, 0, loc,
		)

		effectiveStart := sessionStart
		if from.After(effectiveStart) {
			effectiveStart = from
		}

		effectiveEnd := sessionEnd
		if to.Before(effectiveEnd) {
			effectiveEnd = to
		}

		if !effectiveStart.Before(effectiveEnd) {
			day = day.AddDate(0, 0, 1)
			continue
		}

		query := `
		SELECT
			bucket_ts AS ts,
			argMin(ltp, ts) AS open,
			max(ltp)        AS high,
			min(ltp)        AS low,
			argMax(ltp, ts) AS close
		FROM
		(
			SELECT
				ts,
				ltp,
				(?)
				+ ?
				+ intDiv(
					toUnixTimestamp(ts)
					- toUnixTimestamp(?)
					- ?,
					?
				) * ? AS bucket_ts
			FROM options_moneyness
			WHERE
				underlying = ?
				AND expiry = toDate(?)
				AND strike = ?
				AND option_type = ?
				AND ts >= ?
				AND ts < ?
		)
		WHERE
			bucket_ts >= ? + ?
			AND bucket_ts + ? <= ?
		GROUP BY bucket_ts
		ORDER BY bucket_ts
		`

		rows, err := db.Query(
			query,
			sessionStart,
			offsetSeconds,
			sessionStart,
			offsetSeconds,
			*tfSeconds,
			*tfSeconds,
			underlying,
			expiry,
			strike,
			optionType,
			effectiveStart,
			effectiveEnd,
			sessionStart,
			*tfSeconds,
			*tfSeconds,
			effectiveEnd,
		)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var c candle
			if err := rows.Scan(&c.Ts, &c.Open, &c.High, &c.Low, &c.Close); err != nil {
				rows.Close()
				return nil, err
			}
			all = append(all, c)
		}

		rows.Close()
		day = day.AddDate(0, 0, 1)
	}

	out := models.ColumnarOHLC{
		Ts:    make([]time.Time, 0, len(all)),
		Open:  make([]float64, 0, len(all)),
		High:  make([]float64, 0, len(all)),
		Low:   make([]float64, 0, len(all)),
		Close: make([]float64, 0, len(all)),
	}

	for _, c := range all {
		out.Ts = append(out.Ts, c.Ts)
		out.Open = append(out.Open, c.Open)
		out.High = append(out.High, c.High)
		out.Low = append(out.Low, c.Low)
		out.Close = append(out.Close, c.Close)
	}

	return out, nil
}
