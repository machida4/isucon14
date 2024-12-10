package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type chairPostChairsRequest struct {
	Name               string `json:"name"`
	Model              string `json:"model"`
	ChairRegisterToken string `json:"chair_register_token"`
}

type chairPostChairsResponse struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
}

func chairPostChairs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &chairPostChairsRequest{}
	if err := bindJSON(r, req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Model == "" || req.ChairRegisterToken == "" {
		writeError(w, http.StatusBadRequest, errors.New("some of required fields(name, model, chair_register_token) are empty"))
		return
	}

	owner := &Owner{}
	if err := db.GetContext(ctx, owner, "SELECT * FROM owners WHERE chair_register_token = ?", req.ChairRegisterToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, errors.New("invalid chair_register_token"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	chairID := ulid.Make().String()
	accessToken := secureRandomStr(32)

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO chairs (id, owner_id, name, model, is_active, access_token) VALUES (?, ?, ?, ?, ?, ?)",
		chairID, owner.ID, req.Name, req.Model, false, accessToken,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Path:  "/",
		Name:  "chair_session",
		Value: accessToken,
	})

	writeJSON(w, http.StatusCreated, &chairPostChairsResponse{
		ID:      chairID,
		OwnerID: owner.ID,
	})
}

type postChairActivityRequest struct {
	IsActive bool `json:"is_active"`
}

func chairPostActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chair := ctx.Value("chair").(*Chair)

	req := &postChairActivityRequest{}
	if err := bindJSON(r, req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := db.ExecContext(ctx, "UPDATE chairs SET is_active = ? WHERE id = ?", req.IsActive, chair.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type chairPostCoordinateResponse struct {
	RecordedAt int64 `json:"recorded_at"`
}

func myAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// func chairPostCoordinateOrigin(w http.ResponseWriter, r *http.Request) {
// 	ctx := r.Context()
// 	req := &Coordinate{}
// 	if err := bindJSON(r, req); err != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	chair := ctx.Value("chair").(*Chair)

// 	tx, err := db.Beginx()
// 	if err != nil {
// 		writeError(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	defer tx.Rollback()

// 	var dbChair Chair
// 	if err := tx.GetContext(ctx, &dbChair, "SELECT * FROM chairs WHERE id = ? FOR UPDATE", chair.ID); err != nil {
// 		writeError(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	// beforeLocation := &ChairLocation{}
// 	// tx.GetContext(ctx, beforeLocation, `SELECT * FROM chair_locations WHERE chair_id = ? ORDER BY created_at DESC LIMIT 1`, chair.ID)

// 	// chairLocationID := ulid.Make().String()
// 	// if _, err := tx.ExecContext(
// 	// 	ctx,
// 	// 	`INSERT INTO chair_locations (id, chair_id, latitude, longitude) VALUES (?, ?, ?, ?)`,
// 	// 	chairLocationID, chair.ID, req.Latitude, req.Longitude,
// 	// ); err != nil {
// 	// 	writeError(w, http.StatusInternalServerError, err)
// 	// 	return
// 	// }

// 	// location := &ChairLocation{}
// 	// if err := tx.GetContext(ctx, location, `SELECT * FROM chair_locations WHERE id = ?`, chairLocationID); err != nil {
// 	// 	writeError(w, http.StatusInternalServerError, err)
// 	// 	return
// 	// }

// 	if dbChair.Latitude.Valid && dbChair.Longitude.Valid {
// 		if _, err := tx.ExecContext(
// 			ctx,
// 			`UPDATE chairs SET total_distance = total_distance + ?, total_distance_updated_at = CURRENT_TIMESTAMP(6), latitude = ?, longitude = ? WHERE id = ?`,
// 			myAbs(int(dbChair.Latitude.Int64)-req.Latitude)+myAbs(int(dbChair.Longitude.Int64)-req.Longitude),
// 			req.Latitude,
// 			req.Longitude,
// 			chair.ID,
// 		); err != nil {
// 			writeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 	} else {
// 		if _, err := tx.ExecContext(
// 			ctx,
// 			`UPDATE chairs SET latitude = ?, longitude = ? WHERE id = ?`,
// 			req.Latitude,
// 			req.Longitude,
// 			chair.ID,
// 		); err != nil {
// 			writeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 	}

// 	ride := &Ride{}
// 	if err := tx.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id = ? ORDER BY updated_at DESC LIMIT 1`, chair.ID); err != nil {
// 		if !errors.Is(err, sql.ErrNoRows) {
// 			writeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 	} else {
// 		status, err := getLatestRideStatus(ctx, tx, ride.ID)
// 		if err != nil {
// 			writeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 		if status != "COMPLETED" && status != "CANCELED" {
// 			if req.Latitude == ride.PickupLatitude && req.Longitude == ride.PickupLongitude && status == "ENROUTE" {
// 				if _, err := tx.ExecContext(ctx, "INSERT INTO ride_statuses (id, ride_id, status) VALUES (?, ?, ?)", ulid.Make().String(), ride.ID, "PICKUP"); err != nil {
// 					writeError(w, http.StatusInternalServerError, err)
// 					return
// 				}
// 			}

// 			if req.Latitude == ride.DestinationLatitude && req.Longitude == ride.DestinationLongitude && status == "CARRYING" {
// 				if _, err := tx.ExecContext(ctx, "INSERT INTO ride_statuses (id, ride_id, status) VALUES (?, ?, ?)", ulid.Make().String(), ride.ID, "ARRIVED"); err != nil {
// 					writeError(w, http.StatusInternalServerError, err)
// 					return
// 				}
// 			}
// 		}
// 	}

// 	if err := tx.Commit(); err != nil {
// 		writeError(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	writeJSON(w, http.StatusOK, &chairPostCoordinateResponse{
// 		RecordedAt: dbChair.TotalDistanceUpdatedAt.Time.UnixMilli(),
// 	})
// }

type coordinateUpdate struct {
	ChairID    string
	Latitude   int
	Longitude  int
	TotalDelta int
}

var (
	updateQueue       = make(chan coordinateUpdate, 1000) // 更新キュー
	updateBatch       = make([]coordinateUpdate, 0, 100)  // バッチ更新用スライス
	updateBatchMutex  = sync.Mutex{}
	updateInterval    = time.Second // 更新間隔
	updateWorkerCount = 1           // ワーカーの数
)

func init() {
	// バッチ更新ワーカーを起動
	for i := 0; i < updateWorkerCount; i++ {
		go processCoordinateUpdates()
	}
}

// バッチ更新を処理するゴルーチン
func processCoordinateUpdates() {
	for {
		time.Sleep(updateInterval)

		updateBatchMutex.Lock()
		if len(updateBatch) > 0 {
			batch := updateBatch
			updateBatch = make([]coordinateUpdate, 0, 100)
			updateBatchMutex.Unlock()

			// データベースにバッチ更新
			err := updateCoordinatesInDB(batch)
			if err != nil {
				log.Printf("Failed to update coordinates: %v", err)
			}
		} else {
			updateBatchMutex.Unlock()
		}
	}
}

// データベースにバッチ更新
func updateCoordinatesInDB(batch []coordinateUpdate) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, update := range batch {
		_, err := tx.Exec(
			`UPDATE chairs 
			 SET latitude = ?, longitude = ?, total_distance = total_distance + ? 
			 WHERE id = ?`,
			update.Latitude, update.Longitude, update.TotalDelta, update.ChairID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// `chairPostCoordinate` の修正版
func chairPostCoordinate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &Coordinate{}
	if err := bindJSON(r, req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chair := ctx.Value("chair").(*Chair)

	// 現在の座標を取得
	var current Chair
	err := db.GetContext(ctx, &current, `SELECT latitude, longitude FROM chairs WHERE id = ?`, chair.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// 移動距離を計算
	totalDelta := 0
	if current.Latitude.Valid && current.Longitude.Valid {
		totalDelta = myAbs(int(current.Latitude.Int64)-req.Latitude) + myAbs(int(current.Longitude.Int64)-req.Longitude)
	}

	// 更新データをキューに追加
	updateQueue <- coordinateUpdate{
		ChairID:    chair.ID,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		TotalDelta: totalDelta,
	}

	// レスポンスを即時返却
	writeJSON(w, http.StatusOK, &chairPostCoordinateResponse{
		RecordedAt: time.Now().UnixMilli(),
	})
}

type simpleUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type chairGetNotificationResponse struct {
	Data         *chairGetNotificationResponseData `json:"data"`
	RetryAfterMs int                               `json:"retry_after_ms"`
}

type chairGetNotificationResponseData struct {
	RideID                string     `json:"ride_id"`
	User                  simpleUser `json:"user"`
	PickupCoordinate      Coordinate `json:"pickup_coordinate"`
	DestinationCoordinate Coordinate `json:"destination_coordinate"`
	Status                string     `json:"status"`
}

func chairGetNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chair := ctx.Value("chair").(*Chair)

	ride := &Ride{}
	yetSentRideStatus := RideStatus{}
	status := ""

	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id = ? ORDER BY updated_at DESC LIMIT 1`, chair.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, &chairGetNotificationResponse{
				RetryAfterMs: getRetryAfterMs(),
			})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := db.GetContext(ctx, &yetSentRideStatus, `SELECT * FROM ride_statuses WHERE ride_id = ? AND chair_sent_at IS NULL ORDER BY created_at ASC LIMIT 1`, ride.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status, err = getLatestRideStatus(ctx, db, ride.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		status = yetSentRideStatus.Status
	}

	user := &User{}
	err := db.GetContext(ctx, user, "SELECT * FROM users WHERE id = ? FOR SHARE", ride.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	if yetSentRideStatus.ID != "" {
		_, err := tx.ExecContext(ctx, `UPDATE ride_statuses SET chair_sent_at = CURRENT_TIMESTAMP(6) WHERE id = ?`, yetSentRideStatus.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, &chairGetNotificationResponse{
		Data: &chairGetNotificationResponseData{
			RideID: ride.ID,
			User: simpleUser{
				ID:   user.ID,
				Name: fmt.Sprintf("%s %s", user.Firstname, user.Lastname),
			},
			PickupCoordinate: Coordinate{
				Latitude:  ride.PickupLatitude,
				Longitude: ride.PickupLongitude,
			},
			DestinationCoordinate: Coordinate{
				Latitude:  ride.DestinationLatitude,
				Longitude: ride.DestinationLongitude,
			},
			Status: status,
		},
		RetryAfterMs: getRetryAfterMs(),
	})
}

type postChairRidesRideIDStatusRequest struct {
	Status string `json:"status"`
}

func chairPostRideStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rideID := r.PathValue("ride_id")

	chair := ctx.Value("chair").(*Chair)

	req := &postChairRidesRideIDStatusRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	ride := &Ride{}
	if err := tx.GetContext(ctx, ride, "SELECT * FROM rides WHERE id = ? FOR UPDATE", rideID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("ride not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if ride.ChairID.String != chair.ID {
		writeError(w, http.StatusBadRequest, errors.New("not assigned to this ride"))
		return
	}
	status2 := ""

	switch req.Status {
	// Acknowledge the ride
	case "ENROUTE":
		if _, err := tx.ExecContext(ctx, "INSERT INTO ride_statuses (id, ride_id, status) VALUES (?, ?, ?)", ulid.Make().String(), ride.ID, "ENROUTE"); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		status2 = "ENROUTE"
	// After Picking up user
	case "CARRYING":
		status, err := getLatestRideStatus(ctx, tx, ride.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if status != "PICKUP" {
			writeError(w, http.StatusBadRequest, errors.New("chair has not arrived yet"))
			return
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO ride_statuses (id, ride_id, status) VALUES (?, ?, ?)", ulid.Make().String(), ride.ID, "CARRYING"); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		status2 = "CARRYING"
	default:
		writeError(w, http.StatusBadRequest, errors.New("invalid status"))
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if status2 != "" {
		cache.Set([]byte("latest.ride."+rideID), []byte(status2), 10)
	}

	w.WriteHeader(http.StatusNoContent)
}
