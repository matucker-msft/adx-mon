package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/Azure/adx-mon/ingestor/storage"
	"github.com/Azure/adx-mon/pkg/prompb"
	"github.com/Azure/adx-mon/pkg/wal"
	"github.com/stretchr/testify/require"
)

func TestSeriesKey(t *testing.T) {
	tests := []struct {
		Database []byte
		Labels   []prompb.Label
		Expect   []byte
	}{
		{
			Database: []byte("adxmetrics"),
			Labels:   newTimeSeries("foo", nil, 0, 0).Labels,
			Expect:   []byte("adxmetrics_Foo"),
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.Expect), func(t *testing.T) {
			b := make([]byte, 256)
			require.Equal(t, string(tt.Expect), string(storage.SegmentKey(b[:0], tt.Database, tt.Labels)))
		})
	}
}

func TestStore_Open(t *testing.T) {
	b := make([]byte, 256)
	database := "adxmetrics"
	ctx := context.Background()
	dir := t.TempDir()
	s := storage.NewLocalStore(storage.StoreOpts{
		StorageDir:     dir,
		SegmentMaxSize: 1024,
		SegmentMaxAge:  time.Minute,
	})

	require.NoError(t, s.Open(context.Background()))
	defer s.Close()
	require.Equal(t, 0, s.WALCount())

	ts := newTimeSeries("foo", nil, 0, 0)
	w, err := s.GetWAL(ctx, storage.SegmentKey(b[:0], []byte(database), ts.Labels))
	require.NoError(t, err)
	require.NotNil(t, w)
	require.NoError(t, s.WriteTimeSeries(context.Background(), database, []prompb.TimeSeries{ts}))

	ts = newTimeSeries("foo", nil, 1, 1)
	w, err = s.GetWAL(ctx, storage.SegmentKey(b[:0], []byte(database), ts.Labels))
	require.NoError(t, err)
	require.NotNil(t, w)
	require.NoError(t, s.WriteTimeSeries(context.Background(), database, []prompb.TimeSeries{ts}))

	ts = newTimeSeries("bar", nil, 0, 0)
	w, err = s.GetWAL(ctx, storage.SegmentKey(b[:0], []byte(database), ts.Labels))
	require.NoError(t, err)
	require.NotNil(t, w)
	require.NoError(t, s.WriteTimeSeries(context.Background(), database, []prompb.TimeSeries{ts}))

	path := w.Path()

	require.Equal(t, 2, s.WALCount())
	require.NoError(t, s.Close())

	r, err := wal.NewSegmentReader(path)
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	println(string(data))

	s = storage.NewLocalStore(storage.StoreOpts{
		StorageDir:     dir,
		SegmentMaxSize: 1024,
		SegmentMaxAge:  time.Minute,
	})

	require.NoError(t, s.Open(context.Background()))
	defer s.Close()
	require.Equal(t, 2, s.WALCount())
}

func TestLocalStore_WriteTimeSeries(t *testing.T) {
	b := make([]byte, 256)
	database := "adxmetrics"
	ctx := context.Background()
	dir := t.TempDir()
	s := storage.NewLocalStore(storage.StoreOpts{
		StorageDir:     dir,
		SegmentMaxSize: 1024,
		SegmentMaxAge:  time.Minute,
	})

	require.NoError(t, s.Open(context.Background()))
	defer s.Close()
	require.Equal(t, 0, s.WALCount())

	ts := newTimeSeries("foo", nil, 0, 0)
	w, err := s.GetWAL(ctx, storage.SegmentKey(b[:0], []byte(database), ts.Labels))
	require.NoError(t, err)
	require.NotNil(t, w)
	require.NoError(t, s.WriteTimeSeries(context.Background(), database, []prompb.TimeSeries{ts}))

	path := w.Path()

	require.Equal(t, 1, s.WALCount())
	require.NoError(t, s.Close())

	r, err := wal.NewSegmentReader(path)
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "1970-01-01T00:00:00Z,-414304664621325809,\"{}\",0.000000000\n", string(data))
}

func TestStore_SkipNonCSV(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStore(storage.StoreOpts{
		StorageDir:     dir,
		SegmentMaxSize: 1024,
		SegmentMaxAge:  time.Minute,
	})

	f, err := os.Create(filepath.Join(dir, "foo.csv.gz.tmp"))
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, s.Open(context.Background()))
	defer s.Close()
	require.Equal(t, 0, s.WALCount())
}

func BenchmarkSegmentKey(b *testing.B) {
	buf := make([]byte, 256)
	database := []byte("adxmetrics")
	labels := newTimeSeries("foo", nil, 0, 0).Labels
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SegmentKey(buf[:0], database, labels)
	}
}

func newTimeSeries(name string, labels map[string]string, ts int64, val float64) prompb.TimeSeries {
	l := []prompb.Label{
		{
			Name:  []byte("__name__"),
			Value: []byte(name),
		},
	}
	for k, v := range labels {
		l = append(l, prompb.Label{Name: []byte(k), Value: []byte(v)})
	}
	sort.Slice(l, func(i, j int) bool {
		return bytes.Compare(l[i].Name, l[j].Name) < 0
	})

	return prompb.TimeSeries{
		Labels: l,

		Samples: []prompb.Sample{
			{
				Timestamp: ts,
				Value:     val,
			},
		},
	}
}
