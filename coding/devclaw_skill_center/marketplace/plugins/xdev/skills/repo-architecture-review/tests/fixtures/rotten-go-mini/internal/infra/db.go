package infra

type DB struct {
	// Dummy DB connection
}

func ReadDB() *DB {
	return &DB{}
}

func (db *DB) Query(query string, args ...interface{}) string {
	// Dummy implementation
	return ""
}
