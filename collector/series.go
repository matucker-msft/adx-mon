package collector

import (
	"sort"

	"github.com/Azure/adx-mon/pkg/prompb"
	"github.com/prometheus/client_model/go"
)

type seriesCreator struct {
	AddLabels  map[string]string
	DropLabels map[string]struct{}
}

func (s *seriesCreator) newSeries(name string, scrapeTarget ScrapeTarget, m *io_prometheus_client.Metric) prompb.TimeSeries {
	ts := prompb.TimeSeries{}

	if scrapeTarget.Namespace != "" {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  []byte("adxmon_namespace"),
			Value: []byte(scrapeTarget.Namespace),
		})
	}

	if scrapeTarget.Pod != "" {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  []byte("adxmon_pod"),
			Value: []byte(scrapeTarget.Pod),
		})
	}

	if scrapeTarget.Container != "" {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  []byte("adxmon_container"),
			Value: []byte(scrapeTarget.Container),
		})
	}

	for _, l := range m.Label {
		// Skip labels that will be overridden by static labels
		if _, ok := s.AddLabels[l.GetName()]; ok {
			continue
		}

		// Skip labels that will be dropped
		if _, ok := s.DropLabels[l.GetName()]; ok {
			continue
		}

		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  []byte(l.GetName()),
			Value: []byte(l.GetValue()),
		})
	}

	for k, v := range s.AddLabels {
		if k == "adxmon_namespace" || k == "adxmon_pod" || k == "adxmon_container" {
			continue
		}

		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  []byte(k),
			Value: []byte(v),
		})
	}
	sort.Slice(ts.Labels, func(i, j int) bool {
		return string(ts.Labels[i].Name) < string(ts.Labels[j].Name)
	})

	// Ensure that the __name__ label is the first label
	ts.Labels = append([]prompb.Label{{Name: []byte("__name__"), Value: []byte(name)}}, ts.Labels...)

	return ts
}
