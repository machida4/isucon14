package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
)

func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	rides := []Ride{}

	chairs := []Chair{}
	if err := db.SelectContext(ctx, &chairs, "SELECT * FROM chairs c WHERE c.is_active = TRUE"); err != nil {

	}

	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NULL AND latitude IS NOT NULL ORDER BY created_at`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	for _, v := range rides {
		sort.Slice(chairs, func(i, j int) bool {
			return calculateDistance(v.PickupLatitude, v.PickupLongitude, int(chairs[i].Latitude.Int64), int(chairs[i].Longitude.Int64)) < calculateDistance(v.PickupLatitude, v.PickupLongitude, int(chairs[j].Latitude.Int64), int(chairs[j].Longitude.Int64))
		})
		var i int
		var matched bool
		if i, matched = matchRide(ctx, v, chairs); !matched {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		chairs = append(chairs[:i], chairs[i+1:]...)
	}
	w.WriteHeader(http.StatusNoContent)
}

// rideを受け取って、マッチングさせる。マッチングできたらtrueを返す
func matchRide(ctx context.Context, ride Ride, chairs []Chair) (int, bool) {
	for i, chair := range chairs {
		if isChairEmpty(ctx, chair) {
			if err := saveMatchedRide(ctx, ride, chair); err != nil {
				return i, false
			}
			return i, true
		}
	}

	return len(chairs), false
}

func isChairEmpty(ctx context.Context, chair Chair) bool {
	empty := false
	if err := db.GetContext(ctx, &empty, "SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE", chair.ID); err != nil {
		return false
	}
	return empty
}

func saveMatchedRide(ctx context.Context, ride Ride, chair Chair) error {
	_, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", chair.ID, ride.ID)
	return err
}
