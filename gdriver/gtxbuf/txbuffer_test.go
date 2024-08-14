package gtxbuf_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/rollchains/gordian/gdriver/gtxbuf"
	"github.com/rollchains/gordian/internal/gtest"
	"github.com/stretchr/testify/require"
)

func TestBuffer_Buffered(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := gtxbuf.New(ctx, gtest.NewLogger(t), AddCounterTx, DeleteCounterTx)
	defer buf.Wait()
	defer cancel()

	require.True(t, buf.Initialize(ctx, new(CounterState)))

	// Initializes to empty.
	require.Empty(t, buf.Buffered(ctx, nil))

	// Adding a transaction, shows up.
	require.NoError(t, buf.AddTx(ctx, 1))
	require.Equal(t, []int{1}, buf.Buffered(ctx, nil))

	// Adding another transaction, shows up at the end of the Buffered result.
	require.NoError(t, buf.AddTx(ctx, 2))
	require.Equal(t, []int{1, 2}, buf.Buffered(ctx, nil))

	// And one more transaction, this time explicitly with a destination slice.
	arr := [3]int{}
	require.NoError(t, buf.AddTx(ctx, 3))
	require.Equal(t, []int{1, 2, 3}, buf.Buffered(ctx, arr[:0]))
	require.Equal(t, []int{1, 2, 3}, arr[:])
}

func TestBuffer_Initialize_twicePanics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := gtxbuf.New(ctx, gtest.NewLogger(t), AddCounterTx, DeleteCounterTx)
	defer buf.Wait()
	defer cancel()

	require.True(t, buf.Initialize(ctx, new(CounterState)))

	// Any other operation to avoid a data race on the init channel.
	_ = buf.Buffered(ctx, nil)

	require.Panics(t, func() {
		_ = buf.Initialize(ctx, new(CounterState))
	})
}

func TestBuffer_Rebase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := gtxbuf.New(ctx, gtest.NewLogger(t), AddCounterTx, DeleteCounterTx)
	defer buf.Wait()
	defer cancel()

	require.True(t, buf.Initialize(ctx, new(CounterState)))

	require.NoError(t, buf.AddTx(ctx, 1))
	require.NoError(t, buf.AddTx(ctx, 2))
	require.NoError(t, buf.AddTx(ctx, 3))

	inv, err := buf.Rebase(ctx, &CounterState{
		Version: 1,
		Total:   6,
	}, []int{2, 4})
	require.NoError(t, err)
	require.Empty(t, inv)

	require.Equal(t, []int{1, 3}, buf.Buffered(ctx, nil))
}

// CounterState is a very simple state to use for testing the TxBuffer.
type CounterState struct {
	// How many updates have occurred.
	Version int

	// The sum of all updates.
	Total int
}

// AddCounterTx is the addTx function to use for tests with the CounterState.
func AddCounterTx(_ context.Context, s *CounterState, add int) (*CounterState, error) {
	if add == 0 {
		// Zero is a special case of a plain error,
		// not wrapped in a TxInvalidError.
		return nil, ErrAddZero
	}

	if add < 0 {
		return nil, gtxbuf.TxInvalidError{
			Err: fmt.Errorf("add must be positive (got %d)", add),
		}
	}

	return &CounterState{
		Version: s.Version + 1,
		Total:   s.Total + add,
	}, nil
}

func DeleteCounterTx(_ context.Context, have *[]int, reject []int) {
	rejectValues := make(map[int]struct{}, len(reject))
	for _, r := range reject {
		rejectValues[r] = struct{}{}
	}

	// Track how many elements were actually deleted.
	// DeleteFunc zeros out the trimmed elements.
	// Because it operates on a plain slice value,
	// it can't actually resize the slice.
	trim := 0
	slices.DeleteFunc(*have, func(n int) bool {
		_, ok := rejectValues[n]
		if ok {
			trim++
		}
		return ok
	})

	// Now overwrite have, to exclude the zeroed out elements
	// that were a result of slices.DeleteFunc.
	*have = (*have)[:len(*have)-trim]
}

var ErrAddZero = errors.New("illegal transaction: add 0")
