package testdata

import (
	"embed"
)

//go:embed *sql
var FS embed.FS

const OutboxTestConnUrlTemplate = "postgres://postgres:postgres@localhost:%s/outbox?sslmode=disable"
