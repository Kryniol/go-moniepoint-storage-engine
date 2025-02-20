package domain

import "time"

type IndexEntry struct {
	Key           string
	FileID        string
	ValueSize     int64
	ValuePosition int64
	Timestamp     time.Time
}
