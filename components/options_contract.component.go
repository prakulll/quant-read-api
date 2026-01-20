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
) (interface{}, error) {

	db := services.GetClickHouse()

	// =========================
	// RAW PATH (NO RESAMPLING)
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
			  AND ts BETWEEN ? AND ?
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

		out := make([]models.OptionContractRow, 0)

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

	// =========================================
	// RESAMPLED PATH (MARKET SAFE, FULL CANDLES)
	// =========================================
	query := `
	WITH
		toDateTime(?) AS session_start,
		toDateTime(?) AS session_end,
		? AS tf_seconds,
		? AS offset_seconds
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
			session_start
			+ offset_seconds
			+ intDiv(
				toUnixTimestamp(ts)
				- toUnixTimestamp(session_start)
				- offset_seconds,
				tf_seconds
			) * tf_seconds AS bucket_ts
		FROM options_moneyness
		WHERE underlying = ?
		  AND expiry = toDate(?)
		  AND strike = ?
		  AND option_type = ?
		  AND ts >= session_start
		  AND ts < session_end
	)
	WHERE bucket_ts + tf_seconds <= session_end
	GROUP BY bucket_ts
	ORDER BY bucket_ts
	`

	rows, err := db.Query(
		query,
		from,
		to,
		*tfSeconds,
		offsetSeconds,
		underlying,
		expiry,
		strike,
		optionType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := models.ColumnarOHLC{
		Ts:    []time.Time{},
		Open:  []float64{},
		High:  []float64{},
		Low:   []float64{},
		Close: []float64{},
	}

	for rows.Next() {
		var ts time.Time
		var o, h, l, c float64

		if err := rows.Scan(&ts, &o, &h, &l, &c); err != nil {
			return nil, err
		}

		out.Ts = append(out.Ts, ts)
		out.Open = append(out.Open, o)
		out.High = append(out.High, h)
		out.Low = append(out.Low, l)
		out.Close = append(out.Close, c)
	}

	return out, nil
}
