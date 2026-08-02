package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestMetricDataPointCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build func(m pmetric.Metric) pmetric.Metric
		want  int
	}{
		{
			name: "gauge",
			build: func(m pmetric.Metric) pmetric.Metric {
				gauge := m.SetEmptyGauge()
				gauge.DataPoints().AppendEmpty()
				gauge.DataPoints().AppendEmpty()
				return m
			},
			want: 2,
		},
		{
			name: "sum",
			build: func(m pmetric.Metric) pmetric.Metric {
				sum := m.SetEmptySum()
				sum.DataPoints().AppendEmpty()
				return m
			},
			want: 1,
		},
		{
			name: "histogram",
			build: func(m pmetric.Metric) pmetric.Metric {
				histogram := m.SetEmptyHistogram()
				histogram.DataPoints().AppendEmpty()
				histogram.DataPoints().AppendEmpty()
				histogram.DataPoints().AppendEmpty()
				return m
			},
			want: 3,
		},
		{
			name: "exponential histogram",
			build: func(m pmetric.Metric) pmetric.Metric {
				expHistogram := m.SetEmptyExponentialHistogram()
				expHistogram.DataPoints().AppendEmpty()
				return m
			},
			want: 1,
		},
		{
			name: "summary",
			build: func(m pmetric.Metric) pmetric.Metric {
				summary := m.SetEmptySummary()
				summary.DataPoints().AppendEmpty()
				return m
			},
			want: 1,
		},
		{
			name:  "empty",
			build: func(m pmetric.Metric) pmetric.Metric { return m },
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, metricDataPointCount(tt.build(pmetric.NewMetric())))
		})
	}
}

func TestExemplarsToRaw(t *testing.T) {
	t.Parallel()

	exemplars := pmetric.NewExemplarSlice()
	exemplar := exemplars.AppendEmpty()
	exemplar.SetTimestamp(1234567890)
	exemplar.SetIntValue(42)
	exemplar.SetTraceID([16]byte{1, 2, 3})
	exemplar.SetSpanID([8]byte{4, 5, 6})
	exemplar.FilteredAttributes().PutStr("filtered_key", "filtered_value")

	raw := exemplarsToRaw(exemplars)

	assert.Len(t, raw, 1)

	rawExemplar, ok := raw[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, int64(42), rawExemplar["value"])
	assert.Equal(t, "01020300000000000000000000000000", rawExemplar["trace_id"])
	assert.Equal(t, "0405060000000000", rawExemplar["span_id"])
	assert.Equal(t, map[string]any{"filtered_key": "filtered_value"}, rawExemplar["filtered_attributes"])
}

func TestExemplarsToRawDouble(t *testing.T) {
	t.Parallel()

	exemplars := pmetric.NewExemplarSlice()
	exemplar := exemplars.AppendEmpty()
	exemplar.SetDoubleValue(1.5)

	raw := exemplarsToRaw(exemplars)

	rawExemplar := raw[0].(map[string]any)
	assert.Equal(t, 1.5, rawExemplar["value"])
}

func TestQuantileValuesToRaw(t *testing.T) {
	t.Parallel()

	values := pmetric.NewSummaryDataPointValueAtQuantileSlice()
	value := values.AppendEmpty()
	value.SetQuantile(0.5)
	value.SetValue(10)

	raw := quantileValuesToRaw(values)

	assert.Len(t, raw, 1)
	assert.Equal(t, map[string]any{"quantile": 0.5, "value": 10.0}, raw[0])
}
