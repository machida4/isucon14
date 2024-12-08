// Package models contains the types for schema 'public'.
package models

// Code generated by xo. DO NOT EDIT.

// Connections represents a row from '[custom connections]'.
type Connections struct {
	State string // state
	Count int64  // count
}

// GetConnections runs a custom query, returning results as Connections.
func GetConnections(db XODB) ([]*Connections, error) {
	var err error

	// sql query
	sqlstr := `SELECT state, count(*) ` +
		`FROM pg_stat_activity ` +
		`GROUP BY 1`

	// run query
	XOLog(sqlstr)
	q, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	// load results
	res := []*Connections{}
	for q.Next() {
		c := Connections{}

		// scan
		err = q.Scan(&c.State, &c.Count)
		if err != nil {
			return nil, err
		}

		res = append(res, &c)
	}

	return res, nil
}
