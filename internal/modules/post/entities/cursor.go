package entities

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// ErrCursorMalformed is returned by DecodeCursor for any input that is not
// a cursor this package produced.
var ErrCursorMalformed = errors.New("cursor is malformed")

// cursorFieldSeparator joins the two parts of the cursor before encoding.
// createdAt alone is not unique (two posts can share a millisecond), so the
// cursor is a composite of (createdAt, id) - see §3.4 of the brief - keeping
// the total order the (userId, createdAt desc) index already produces.
const cursorFieldSeparator = "|"

// Cursor identifies the last item a caller saw, so the next page can
// continue from exactly there instead of using skip/offset (§3.4: skip
// walks and discards n documents, so cost grows with page depth).
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// Encode produces the opaque, URL-safe string a client passes back as
// ?cursor=.
func Encode(c Cursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + cursorFieldSeparator + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Decode parses a cursor string produced by Encode. An empty string decodes
// to the zero Cursor with no error - core.go treats that as "first page".
func Decode(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrCursorMalformed
	}

	parts := strings.SplitN(string(raw), cursorFieldSeparator, 2)
	if len(parts) != 2 || parts[1] == "" {
		return Cursor{}, ErrCursorMalformed
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, ErrCursorMalformed
	}

	return Cursor{CreatedAt: createdAt, ID: parts[1]}, nil
}
