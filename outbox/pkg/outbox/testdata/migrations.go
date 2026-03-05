package testdata

import (
	"embed"
)

//go:embed *sql
var FS embed.FS

const BookingsConnUrlTemplate = "postgres://postgres:postgres@localhost:%s/bookings?sslmode=disable"
