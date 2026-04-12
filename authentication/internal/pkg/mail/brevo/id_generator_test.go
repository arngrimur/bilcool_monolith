package brevo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextIncrementsIdCounter(t *testing.T) {
	generator := IDGenerator{}

	id1 := generator.NextID()
	id2 := generator.NextID()

	require.Equal(t, id1, 1)
	require.Equal(t, id2, 2)
	require.Equal(t, generator.idCounter, 2)
}
