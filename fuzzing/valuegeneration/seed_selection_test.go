package valuegeneration

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/stretchr/testify/require"
)

func seedSelectionGenerator(size int) *MutationalValueGenerator {
	values := NewValueSet()
	for i := 0; i < size; i++ {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(i))
		values.AddBytes(data)
		values.AddString(fmt.Sprintf("seed-%d-世界", i))
		values.AddAddress(common.BytesToAddress(data))
	}
	return NewMutationalValueGenerator(&MutationalValueGeneratorConfig{
		MutateBytesProbability: 1, MutateStringProbability: 1,
		RandomValueGeneratorConfig: &RandomValueGeneratorConfig{
			GenerateRandomBytesMinSize: 8, GenerateRandomBytesMaxSize: 8,
			GenerateRandomStringMinSize: 8, GenerateRandomStringMaxSize: 8,
		},
	}, values, rand.New(rand.NewSource(1)))
}

func TestSeedSelection(t *testing.T) {
	t.Parallel()
	g := seedSelectionGenerator(32)
	seenBytes, seenStrings, seenAddresses := map[string]bool{}, map[string]bool{},
		map[common.Address]bool{}
	for i := 0; i < 4096; i++ {
		data, text, address := g.GenerateBytes(), g.GenerateString(), g.GenerateAddress()
		require.True(t, g.valueSet.ContainsBytes(data))
		require.True(t, g.valueSet.ContainsString(text))
		require.True(t, g.valueSet.ContainsAddress(address))
		seenBytes[string(data)], seenStrings[text], seenAddresses[address] = true, true, true
		original := slices.Clone(data)
		data[0] ^= 0xff
		require.True(t, g.valueSet.ContainsBytes(original), "output must not alias seed data")
	}
	require.Len(t, seenBytes, 32)
	require.Len(t, seenStrings, 32)
	require.Len(t, seenAddresses, 32)
	for _, seed := range g.valueSet.Bytes() {
		require.True(t, g.valueSet.ContainsBytes(seed))
	}
}

func TestSeedMutationPreservesInputs(t *testing.T) {
	t.Parallel()
	g := seedSelectionGenerator(16)
	g.config.MaxMutationRounds = 4
	input := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	for i := 0; i < 512; i++ {
		g.MutateBytes(input)
		fixed := g.MutateFixedBytes(input)
		require.Len(t, fixed, len(input))
		require.Equal(t, []byte{0, 1, 2, 3, 4, 5, 6, 7}, input)
		g.MutateString("input-世界")
	}
	for _, seed := range g.valueSet.Bytes() {
		require.True(t, g.valueSet.ContainsBytes(seed), "seed content must still match its key")
	}
}

func TestSeedSelectionFallbacks(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1} {
		g := seedSelectionGenerator(size)
		// Empty sets must fall back even with zero random bias; populated sets obey a bias of one.
		g.config.GenerateRandomBytesBias = float32(size)
		g.config.GenerateRandomStringBias = float32(size)
		g.config.GenerateRandomAddressBias = float32(size)
		for i := 0; i < 32; i++ {
			require.Len(t, g.GenerateBytes(), 8)
			require.Len(t, g.GenerateString(), 8)
			require.Len(t, g.GenerateFixedBytes(i+1), i+1)
			g.GenerateAddress()
		}
	}
	for _, seed := range [][]byte{nil, {}} {
		g := seedSelectionGenerator(0)
		g.valueSet.AddBytes(seed)
		require.Equal(t, seed, g.GenerateBytes())
		require.Equal(t, []byte{0, 0}, g.GenerateFixedBytes(2))
		g.valueSet.AddString("")
		require.Empty(t, g.GenerateString())
	}
}

// BenchmarkSeedSelection measures seeded generation, random fallback, and supplied-input mutation.
func BenchmarkSeedSelection(b *testing.B) {
	for _, size := range []int{8, 256} {
		for _, mode := range []string{"seeded", "random", "existing"} {
			b.Run(fmt.Sprintf("seeds=%d/mode=%s", size, mode), func(b *testing.B) {
				g := seedSelectionGenerator(size)
				operations := map[string]func(){
					"bytes":   func() { g.GenerateBytes() },
					"string":  func() { g.GenerateString() },
					"address": func() { g.GenerateAddress() },
				}
				if mode == "random" {
					g.config.GenerateRandomBytesBias = 1
					g.config.GenerateRandomStringBias = 1
					g.config.GenerateRandomAddressBias = 1
				}
				if mode == "existing" {
					data := []byte("input-01")
					operations["bytes"] = func() { g.MutateBytes(data) }
					operations["string"] = func() { g.MutateString("input-世界") }
					delete(operations, "address")
				}
				for _, name := range []string{"bytes", "string", "address"} {
					if operation, ok := operations[name]; ok {
						b.Run(name, func(b *testing.B) {
							b.ReportAllocs()
							for i := 0; i < b.N; i++ {
								operation()
							}
						})
					}
				}
			})
		}
	}
}
