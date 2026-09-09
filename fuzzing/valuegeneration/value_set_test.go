package valuegeneration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapEntryAtBounds(t *testing.T) {
	t.Parallel()
	entries := map[int]string{1: "one", 2: "two"}
	for index := 0; index < len(entries); index++ {
		key, value := mapEntryAt(entries, index)
		require.Equal(t, entries[key], value)
	}
	for _, index := range []int{-1, 2} {
		require.PanicsWithValue(t, "value set index out of range", func() {
			mapEntryAt(entries, index)
		})
	}
	require.PanicsWithValue(t, "value set index out of range", func() {
		mapEntryAt(map[int]string(nil), 0)
	})
}
