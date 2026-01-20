package components

import (
	"time"

	"quant-read-api/models"
	"quant-read-api/services"
)

func GetFuturesData(
	underlying string,
	series string,
	from time.Time,
	to time.Time,
	tfSeconds *int64,
	offsetSeconds int64,
) (any, error) {

	db := services.GetClickHouse()

	// =========================
	// RAW PATH
	// =========================
	if tfSeconds == nil {
		query := `
			SELECT
				ts,
				futures_price,
				underlying,
				series
			FROM second_data.futures_data
			WHERE underlying = ?
			  AND series = ?
			  AND ts >= ?
			  AND ts < ?
			ORDER BY ts
		`

		rows, err := db.Query(query, underlying, series, from, to)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		out := make([]models.FuturesDataRow, 0)

		for rows.Next() {
			var r models.FuturesDataRow
			if err := rows.Scan(&r.Ts, &r.FuturesPrice, &r.Underlying, &r.Series); err != nil {
				return nil, err
			}
			out = append(out, r)
		}

		return out, nil
	}

	// =========================
	// RESAMPLED PATH
	// =========================

	loc, _ := time.LoadLocation("Asia/Kolkata")
	tradeDate := from.In(loc)

	sessionStart := time.Date(
		tradeDate.Year(), tradeDate.Month(), tradeDate.Day(),
		9, 15, 0, 0, loc,
	)
	sessionEnd := time.Date(
		tradeDate.Year(), tradeDate.Month(), tradeDate.Day(),
		15, 30, 0, 0, loc,
	)

	query := `
	WITH
		toDateTime(?) AS session_start,
		toDateTime(?) AS session_end,
		toDateTime(?) AS user_from,
		toDateTime(?) AS user_to,
		? AS tf_seconds,
		? AS offset_seconds,
		greatest(session_start, user_from) AS effective_start,
		least(session_end, user_to) AS effective_end
	SELECT
		bucket_ts AS ts,
		argMin(futures_price, ts) AS open,
		max(futures_price)        AS high,
		min(futures_price)        AS low,
		argMax(futures_price, ts) AS close
	FROM
	(
		SELECT
			ts,
			futures_price,
			session_start
			+ offset_seconds
			+ intDiv(
				toUnixTimestamp(ts)
				- toUnixTimestamp(session_start)
				- offset_seconds,
				tf_seconds
			) * tf_seconds AS bucket_ts
		FROM second_data.futures_data
		WHERE underlying = ?
		  AND series = ?
		  AND ts >= effective_start
		  AND ts < effective_end
	)
	WHERE
		bucket_ts >= session_start + tf_seconds
		AND bucket_ts + tf_seconds <= effective_end
	GROUP BY bucket_ts
	ORDER BY bucket_ts
	`

	rows, err := db.Query(
		query,
		sessionStart,
		sessionEnd,
		from,
		to,
		*tfSeconds,
		offsetSeconds,
		underlying,
		series,
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
