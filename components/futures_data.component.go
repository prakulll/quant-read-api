package components

import (
	"sync"
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
			if err := rows.Scan(
				&r.Ts,
				&r.FuturesPrice,
				&r.Underlying,
				&r.Series,
			); err != nil {
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

	// Normalize to date boundaries
	startDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	endDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc)

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

		// Clamp to user range
		effectiveStart := sessionStart
		if from.After(effectiveStart) {
			effectiveStart = from
		}

		effectiveEnd := sessionEnd
		if to.Before(effectiveEnd) {
			effectiveEnd = to
		}

		if !effectiveStart.Before(effectiveEnd) {
			continue
		}

		query := `
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
				?
				+ ?
				+ intDiv(
					toUnixTimestamp(ts)
					- toUnixTimestamp(?)
					- ?,
					?
				) * ? AS bucket_ts
			FROM second_data.futures_data
			WHERE underlying = ?
			  AND series = ?
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
			series,
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

// ==================================================
// V2: concurrent futures fetch (series fan-out)
// ==================================================
func GetFuturesDataConcurrent(
	underlying string,
	seriesList []string,
	from time.Time,
	to time.Time,
	tfSeconds *int64,
	offsetSeconds int64,
) (map[string]any, error) {

	results := make(map[string]any)
	var mu sync.Mutex
	var wg sync.WaitGroup

	var firstErr error

	for _, s := range seriesList {
		series := s
		wg.Add(1)

		go func() {
			defer wg.Done()

			data, err := GetFuturesData(
				underlying,
				series,
				from,
				to,
				tfSeconds,
				offsetSeconds,
			)
			if err != nil {
				firstErr = err
				return
			}

			mu.Lock()
			results[series] = data
			mu.Unlock()
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}
