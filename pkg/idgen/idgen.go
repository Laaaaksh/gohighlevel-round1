// Package idgen is the single place every module gets a new entity id from.
// It hands out UUIDv7 strings: v7 embeds a millisecond timestamp in its
// high bits, so ids sort roughly by creation time and inserts land at the
// tail of a B-tree index instead of scattering across it the way UUIDv4's
// pure randomness does. At the follows/posts volumes this service targets,
// that avoids the write-amplification and page-split cost random-insert
// ids cause on the primary index of every collection.
package idgen

import "github.com/google/uuid"

// New returns a new UUIDv7, string-encoded. Every collection stores ids in
// this same string form, so a query never has to convert between a driver
// UUID type and a string to compare or index them.
func New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
