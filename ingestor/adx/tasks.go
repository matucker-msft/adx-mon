package adx

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	v1 "github.com/Azure/adx-mon/api/v1"
	"github.com/Azure/adx-mon/pkg/logger"
	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/kql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TableDetail struct {
	TableName       string  `kusto:"TableName"`
	HotExtentSize   float64 `kusto:"HotExtentSize"`
	TotalExtentSize float64 `kusto:"TotalExtentSize"`
	TotalExtents    int64   `kusto:"TotalExtents"`
	HotRowCount     int64   `kusto:"HotRowCount"`
	TotalRowCount   int64   `kusto:"TotalRowCount"`
}

type DropUnusedTablesTask struct {
	mu           sync.Mutex
	unusedTables map[string]int
	kustoCli     StatementExecutor
	database     string
}

type StatementExecutor interface {
	Database() string
	Mgmt(ctx context.Context, query kusto.Statement, options ...kusto.MgmtOption) (*kusto.RowIterator, error)
}

func NewDropUnusedTablesTask(kustoCli StatementExecutor) *DropUnusedTablesTask {
	return &DropUnusedTablesTask{
		unusedTables: make(map[string]int),
		kustoCli:     kustoCli,
	}
}

func (t *DropUnusedTablesTask) Run(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	details, err := t.loadTableDetails(ctx)
	if err != nil {
		return fmt.Errorf("error loading table details: %w", err)
	}

	for _, v := range details {
		if v.TotalRowCount > 0 {
			delete(t.unusedTables, v.TableName)
		}

		if v.TotalRowCount == 0 {
			t.unusedTables[v.TableName]++
			logger.Infof("Marking table %s.%s as unused", t.database, v.TableName)
		}
	}

	for table, count := range t.unusedTables {
		if count > 2 {
			logger.Infof("DRYRUN Dropping unused table %s.%s", t.kustoCli.Database(), table)
			// stmt := kusto.NewStmt("", kusto.UnsafeStmt(unsafe.Stmt{Add: true, SuppressWarning: true})).UnsafeAdd(fmt.Sprintf(".drop table %s", table))
			// if _, err := t.kustoCli.Mgmt(ctx, stmt); err != nil {
			// 	return fmt.Errorf("error dropping table %s: %w", table, err)
			// }
			delete(t.unusedTables, table)
		}
	}
	return nil
}

func (t *DropUnusedTablesTask) loadTableDetails(ctx context.Context) ([]TableDetail, error) {
	stmt := kusto.NewStmt(".show tables details | project TableName, HotExtentSize, TotalExtentSize, TotalExtents, HotRowCount, TotalRowCount")
	rows, err := t.kustoCli.Mgmt(ctx, stmt)
	if err != nil {
		return nil, err
	}

	var tables []TableDetail
	for {
		row, err1, err2 := rows.NextRowOrError()
		if err2 == io.EOF {
			return tables, nil
		} else if err1 != nil {
			return tables, err1
		} else if err2 != nil {
			return tables, err2
		}

		var v TableDetail
		if err := row.ToStruct(&v); err != nil {
			return tables, err
		}
		tables = append(tables, v)
	}
}

type FunctionStore interface {
	Functions() []*v1.Function
	UpdateStatus(ctx context.Context, fn *v1.Function) error
}

type SyncFunctionsTask struct {
	cache map[string]*v1.Function
	mu    sync.RWMutex

	store    FunctionStore
	kustoCli StatementExecutor
}

func NewSyncFunctionsTask(store FunctionStore, kustoCli StatementExecutor) *SyncFunctionsTask {
	return &SyncFunctionsTask{
		cache:    make(map[string]*v1.Function),
		store:    store,
		kustoCli: kustoCli,
	}
}

func (t *SyncFunctionsTask) updateStatus(fn *v1.Function, status v1.FunctionStatusEnum) {
	fn.Status.LastTimeReconciled = metav1.Time{Time: time.Now()}
	fn.Status.Status = status
	if err := t.store.UpdateStatus(context.Background(), fn); err != nil {
		logger.Errorf("Failed to update status for function %s.%s: %v", fn.Spec.Database, fn.Name, err)
	}
}

func (t *SyncFunctionsTask) Run(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	functions := t.store.Functions()
	for _, function := range functions {

		if function.Spec.Database != t.kustoCli.Database() {
			continue
		}

		cacheKey := function.Spec.Database + function.Name
		if fn, ok := t.cache[cacheKey]; ok {
			if function.ResourceVersion != fn.ResourceVersion {
				// invalidate our cache
				delete(t.cache, cacheKey)
			} else {
				// function is up to date
				continue
			}
		}

		stmt := kql.New("").AddUnsafe(function.Spec.Body)
		if _, err := t.kustoCli.Mgmt(ctx, stmt); err != nil {
			if !errors.Retry(err) {
				t.updateStatus(function, v1.PermanentFailure)
				logger.Errorf("Permanent failure to create function %s.%s: %v", function.Spec.Database, function.Name, err)
				// We want to fall through here so that we can cache this object, there's no need
				// to retry creating it. If it's updated, we'll detect the change in the cached
				// object and try again after invalidating the cache.
				t.cache[cacheKey] = function
				continue
			} else {
				t.updateStatus(function, v1.Failed)
				logger.Warnf("Transient failure to create function %s.%s: %v", function.Spec.Database, function.Name, err)
				continue
			}
		}

		logger.Infof("Successfully created function %s.%s", function.Spec.Database, function.Name)
		t.updateStatus(function, v1.Success)
	}

	return nil
}
