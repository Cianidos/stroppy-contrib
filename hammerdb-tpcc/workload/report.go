package workload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/stroppy-io/stroppy/v6/pkg/bench"
)

const transactionCountSQL = "select sum(xact_commit + xact_rollback) from pg_stat_database"
const newOrderCountSQL = "select sum(d_next_o_id) from district"

type measurement struct {
	Reference    string      `json:"reference"`
	Mode         string      `json:"mode"`
	Workers      int         `json:"workers"`
	Clients      int         `json:"clients"`
	Seed         int64       `json:"seed"`
	Valid        bool        `json:"valid"`
	Start        time.Time   `json:"start"`
	End          time.Time   `json:"end"`
	Minutes      float64     `json:"minutes"`
	StartTPM     int64       `json:"startTransactions"`
	EndTPM       int64       `json:"endTransactions"`
	StartNOPM    int64       `json:"startNewOrders"`
	EndNOPM      int64       `json:"endNewOrders"`
	NOPM         float64     `json:"nopm"`
	TPM          float64     `json:"tpm"`
	ClientCounts [][5]uint64 `json:"clientCounts"`
}

func (w *workload) monitor(ctx context.Context, b *bench.Bench) (err error) {
	session, err := w.connect(ctx, b, "default")
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		err = errors.Join(err, session.Commit(cleanup))
	}()
	if err := sleep(ctx, w.rampup); err != nil {
		return err
	}
	result := measurement{Minutes: w.duration.Minutes()}
	result.StartTPM, err = session.QueryValue[int64](ctx, transactionCountSQL, nil)
	if err != nil {
		return err
	}
	result.StartNOPM, err = session.QueryValue[int64](ctx, newOrderCountSQL, nil)
	if err != nil {
		return err
	}
	result.Start = time.Now()
	if err := sleep(ctx, w.duration); err != nil {
		return err
	}
	result.EndTPM, err = session.QueryValue[int64](ctx, transactionCountSQL, nil)
	if err != nil {
		return err
	}
	result.EndNOPM, err = session.QueryValue[int64](ctx, newOrderCountSQL, nil)
	if err != nil {
		return err
	}
	result.End = time.Now()
	// HammerDB uses integer division for whole-minute windows.
	result.NOPM = perMinute(result.EndNOPM-result.StartNOPM, w.duration)
	result.TPM = perMinute(result.EndTPM-result.StartTPM, w.duration)
	result.Valid = true
	w.result = result
	w.nopm.Set(ctx, result.NOPM)
	w.tpm.Set(ctx, result.TPM)
	return nil
}

func perMinute(count int64, duration time.Duration) float64 {
	if duration%time.Minute == 0 {
		return float64(count / int64(duration/time.Minute))
	}
	return float64(count) / duration.Minutes()
}

func (w *workload) report(bench.ReportContext) (bench.ReportContribution, error) {
	result := w.result
	result.Reference = "HammerDB v6.1 / 2442b32756a484de4206bb0061d9bb28869beaf0"
	result.Mode, result.Workers, result.Seed = w.mode, w.workers, w.seed
	result.Clients = w.workers
	if w.asyncScale {
		result.Clients *= w.asyncClients
	}
	return bench.ReportContribution{Data: result}, nil
}

func renderReport(writer io.Writer, data json.RawMessage) error {
	var result measurement
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if !result.Valid {
		_, err := fmt.Fprintf(writer, "HammerDB TPROC-C: %s mode, no completed timed counter window\n", result.Mode)
		return err
	}
	_, err := fmt.Fprintf(writer, "HammerDB TPROC-C: %.0f NOPM / %.0f PostgreSQL TPM; %d clients, %.3f minutes (database counters, not Stroppy TPS)\n", result.NOPM, result.TPM, result.Clients, result.Minutes)
	return err
}
