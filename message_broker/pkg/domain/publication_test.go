package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPubs(t *testing.T) {
	p := PublicationBase{
		Tables: []string{"foo", "bar"},
	}
	expected := "'foo,bar'"
	result := p.GetPubs()
	require.Equal(t, expected, result)
}
