package components

import (
	"time"

	"quant-read-api/models"
	"quant-read-api/services"
)

func GetIndexData(
	underlying string,
	from time.Time,
	to time.Time,
	tfSeconds *int64,
	offsetSeconds int64,
) (any, error) {

	db := services.GetClickHouse()

	if tfSeconds == nil {
		query := `
			SELECT
				ts,
				spot_price,
				underlying
			FROM second_data.index_data
			WHERE underlying = ?
			  AND ts >= ?
			  AND ts < ?
			ORDER BY ts
		`

		rows, err := db.Query(query, underlying, from, to)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		out := make([]models.IndexDataRow, 0)

		for rows.Next() {
			var r models.IndexDataRow
			if err := rows.Scan(&r.Ts, &r.SpotPrice, &r.Underlying); err != nil {
				return nil, err
			}
			out = append(out, r)
		}

		return out, nil
	}

	loc, _ := time.LoadLocation("Asia/Kolkata")

	out := models.ColumnarOHLC{
		Ts:    []time.Time{},
		Open:  []float64{},
		High:  []float64{},
		Low:   []float64{},
		Close: []float64{},
	}

	// Normalize to start-of-day
	startDate := time.Date(
		from.In(loc).Year(),
		from.In(loc).Month(),
		from.In(loc).Day(),
		0, 0, 0, 0,
		loc,
	)

	endDate := time.Date(
		to.In(loc).Year(),
		to.In(loc).Month(),
		to.In(loc).Day(),
		0, 0, 0, 0,
		loc,
	)

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {

		sessionStart := time.Date(
			d.Year(), d.Month(), d.Day(),
			9, 15, 0, 0,
			loc,
		)

		sessionEnd := time.Date(
			d.Year(), d.Month(), d.Day(),
			15, 30, 0, 0,
			loc,
		)

		// clamp to user range
		effectiveStart := sessionStart
		if effectiveStart.Before(from) {
			effectiveStart = from
		}

		effectiveEnd := sessionEnd
		if effectiveEnd.After(to) {
			effectiveEnd = to
		}

		if !effectiveStart.Before(effectiveEnd) {
			continue
		}

		query := `
		WITH
			toDateTime(?) AS session_start,
			toDateTime(?) AS session_end,
			toDateTime(?) AS effective_start,
			toDateTime(?) AS effective_end,
			? AS tf_seconds,
			? AS offset_seconds
		SELECT
			bucket_ts AS ts,
			argMin(spot_price, ts) AS open,
			max(spot_price)        AS high,
			min(spot_price)        AS low,
			argMax(spot_price, ts) AS close
		FROM
		(
			SELECT
				ts,
				spot_price,
				session_start
				+ offset_seconds
				+ intDiv(
					toUnixTimestamp(ts)
					- toUnixTimestamp(session_start)
					- offset_seconds,
					tf_seconds
				) * tf_seconds AS bucket_ts
			FROM second_data.index_data
			WHERE underlying = ?
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
			effectiveStart,
			effectiveEnd,
			*tfSeconds,
			offsetSeconds,
			underlying,
		)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var ts time.Time
			var o, h, l, c float64

			if err := rows.Scan(&ts, &o, &h, &l, &c); err != nil {
				rows.Close()
				return nil, err
			}

			out.Ts = append(out.Ts, ts)
			out.Open = append(out.Open, o)
			out.High = append(out.High, h)
			out.Low = append(out.Low, l)
			out.Close = append(out.Close, c)
		}

		rows.Close()
	}

	return out, nil
}
