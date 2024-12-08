-- SELECT id,
--        owner_id,
--        name,
--        access_token,
--        model,
--        is_active,
--        created_at,
--        updated_at,
--        IFNULL(total_distance, 0) AS total_distance,
--        total_distance_updated_at
-- FROM chairs
--        LEFT JOIN (SELECT chair_id,
--                           SUM(IFNULL(distance, 0)) AS total_distance,
--                           MAX(created_at)          AS total_distance_updated_at
--                    FROM (SELECT chair_id,
--                                 created_at,
--                                 ABS(latitude - LAG(latitude) OVER (PARTITION BY chair_id ORDER BY created_at)) +
--                                 ABS(longitude - LAG(longitude) OVER (PARTITION BY chair_id ORDER BY created_at)) AS distance
--                          FROM chair_locations) tmp
--                    GROUP BY chair_id) distance_table ON distance_table.chair_id = chairs.id
-- WHERE owner_id = ?;

ALTER TABLE chairs ADD COLUMN total_distance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chairs ADD COLUMN total_distance_updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);
