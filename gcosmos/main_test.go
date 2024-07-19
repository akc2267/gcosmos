package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Cheap toggle at build time to quickly switch to running comet,
// for cases where the difference in behavior needs to be inspected.
const runCometInsteadOfGordian = false

func TestRootCmd(t *testing.T) {
	t.Parallel()

	e := NewRootCmd(t)

	e.Run("init", "defaultmoniker").NoError(t)
}

func TestRootCmd_startWithGordian_singleValidator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := ConfigureChain(t, ctx, ChainConfig{
		ID:            t.Name(),
		NVals:         1,
		StakeStrategy: ConstantStakeStrategy(1_000_000_000),
	})

	var startCmd = []string{"start"}
	var gHTTPAddrFile string
	if !runCometInsteadOfGordian {
		gHTTPAddrFile = filepath.Join(t.TempDir(), "http_addr.txt")
		// Then include the HTTP server flags.
		startCmd = append(
			startCmd,
			"--g-http-addr", "127.0.0.1:0",
			"--g-http-addr-file", gHTTPAddrFile,
		)
	}

	// Ensure the start command has fully completed by the end of the test.
	startDone := make(chan struct{})
	go func() {
		defer close(startDone)
		_ = c.RootCmds[0].RunC(ctx, startCmd...)
	}()
	defer func() {
		<-startDone
	}()
	defer cancel()

	// Get the HTTP address, which may require a few tries,
	// depending on how quickly the start command begins.
	if !runCometInsteadOfGordian {
		// Gratuitously long deadline.
		var httpAddr string
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			a, err := os.ReadFile(gHTTPAddrFile)
			if err != nil {
				// Swallow the error and delay.
				time.Sleep(25 * time.Millisecond)
				continue
			}
			if !bytes.HasSuffix(a, []byte("\n")) {
				// Very unlikely incomplete write/read.
				time.Sleep(25 * time.Millisecond)
				continue
			}

			httpAddr = strings.TrimSuffix(string(a), "\n")
			break
		}

		if httpAddr == "" {
			t.Fatalf("did not read http address from %s in time", gHTTPAddrFile)
		}

		u := "http://" + httpAddr + "/blocks/watermark"
		// TODO: we might need to delay until we get a non-error HTTP response.

		deadline = time.Now().Add(10 * time.Second)
		var maxHeight uint
		for time.Now().Before(deadline) {
			resp, err := http.Get(u)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var m map[string]uint
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
			resp.Body.Close()

			maxHeight = m["VotingHeight"]
			if maxHeight < 3 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// We got at least to height 3, so quit the loop.
			break
		}

		require.GreaterOrEqual(t, maxHeight, uint(3))
	}
}

func TestRootCmd_startWithGordian_multipleValidators(t *testing.T) {
	t.Skip("Not yet ready")

	const totalVals = 11
	const interestingVals = 4

	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := ConfigureChain(t, ctx, ChainConfig{
		ID:    t.Name(),
		NVals: totalVals,

		// Due to an outstanding and undocumented SDK bug,
		// we require at least 11 validators.
		// But, we don't want to run 11 validators.
		// So put the majority of the stake in the first four validators,
		// and the rest get the bare minimum.
		StakeStrategy: func(idx int) string {
			const minAmount = "1000000"
			const largeStake = minAmount + "000000stake"
			if idx < interestingVals {
				// Validators with substantial stake.
				return largeStake
			}

			// Beyond that, give them bare minimum stake.
			return minAmount + "stake"
		},
	})

	httpAddrDir := t.TempDir()
	httpAddrFiles := make([]string, interestingVals)

	// Ensure the start command has fully completed by the end of the test.
	var wg sync.WaitGroup
	wg.Add(interestingVals)
	for i := range interestingVals {
		httpAddrFiles[i] = filepath.Join(httpAddrDir, fmt.Sprintf("http_addr_%d.txt", i))

		go func(i int) {
			defer wg.Done()

			startCmd := []string{"start"}
			if !runCometInsteadOfGordian {
				// Then include the HTTP server flags.
				startCmd = append(
					startCmd,
					"--g-http-addr", "127.0.0.1:0",
					"--g-http-addr-file", httpAddrFiles[i],
				)
			}

			_ = c.RootCmds[i].RunC(ctx, startCmd...)
		}(i)
	}
	defer wg.Wait()
	defer cancel()

	// Each of the interesting validators must report a height beyond the first few blocks.
	for i := range interestingVals {
		if runCometInsteadOfGordian {
			// Nothing to check in this mode.
			break
		}

		// Gratuitously long deadline to confirm the HTTP address.
		var httpAddr string
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			a, err := os.ReadFile(httpAddrFiles[i])
			if err != nil {
				// Swallow the error and delay.
				time.Sleep(25 * time.Millisecond)
				continue
			}
			if !bytes.HasSuffix(a, []byte("\n")) {
				// Very unlikely incomplete write/read.
				time.Sleep(25 * time.Millisecond)
				continue
			}

			httpAddr = strings.TrimSuffix(string(a), "\n")
			break
		}

		if httpAddr == "" {
			t.Fatalf("did not read http address from %s in time", httpAddrFiles[i])
		}

		u := "http://" + httpAddr + "/blocks/watermark"
		// TODO: we might need to delay until we get a non-error HTTP response.

		// Another deadline to confirm the voting height.
		deadline = time.Now().Add(10 * time.Second)
		var maxHeight uint
		for time.Now().Before(deadline) {
			resp, err := http.Get(u)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var m map[string]uint
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
			resp.Body.Close()

			maxHeight = m["VotingHeight"]
			if maxHeight < 3 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// We got at least to height 3, so quit the loop.
			break
		}

		require.GreaterOrEqualf(t, maxHeight, uint(3), "checking max block height on validator at index %d", i)
	}
}