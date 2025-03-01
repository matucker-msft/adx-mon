package schema_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/adx-mon/schema"
	"github.com/stretchr/testify/require"
)

func TestNewSchema_NoLabels(t *testing.T) {
	mapping := schema.NewMetricsSchema()
	_, err := json.Marshal(mapping)
	require.NoError(t, err)

	require.Equal(t, len(mapping), 4)
	require.Equal(t, "Timestamp", mapping[0].Column)
	require.Equal(t, "datetime", mapping[0].DataType)
	require.Equal(t, "0", mapping[0].Properties.Ordinal)

	require.Equal(t, "SeriesId", mapping[1].Column)
	require.Equal(t, "long", mapping[1].DataType)
	require.Equal(t, "1", mapping[1].Properties.Ordinal)

	require.Equal(t, "Labels", mapping[2].Column)
	require.Equal(t, "dynamic", mapping[2].DataType)
	require.Equal(t, "2", mapping[2].Properties.Ordinal)

	require.Equal(t, "Value", mapping[3].Column)
	require.Equal(t, "real", mapping[3].DataType)
	require.Equal(t, "3", mapping[3].Properties.Ordinal)
}

func TestNewSchema_AddConstMapping(t *testing.T) {
	mapping := schema.NewMetricsSchema()
	mapping = mapping.AddConstMapping("Region", "eastus")

	_, err := json.Marshal(mapping)
	require.NoError(t, err)

	require.Equal(t, len(mapping), 5)
	require.Equal(t, "Timestamp", mapping[0].Column)
	require.Equal(t, "datetime", mapping[0].DataType)
	require.Equal(t, "0", mapping[0].Properties.Ordinal)

	require.Equal(t, "SeriesId", mapping[1].Column)
	require.Equal(t, "long", mapping[1].DataType)
	require.Equal(t, "1", mapping[1].Properties.Ordinal)

	require.Equal(t, "Labels", mapping[2].Column)
	require.Equal(t, "dynamic", mapping[2].DataType)
	require.Equal(t, "2", mapping[2].Properties.Ordinal)

	require.Equal(t, "Value", mapping[3].Column)
	require.Equal(t, "real", mapping[3].DataType)
	require.Equal(t, "3", mapping[3].Properties.Ordinal)

	require.Equal(t, "Region", mapping[4].Column)
	require.Equal(t, "string", mapping[4].DataType)
	require.Equal(t, "4", mapping[4].Properties.Ordinal)
	require.Equal(t, "eastus", mapping[4].Properties.ConstValue)

}

func TestNewSchema_AddLiftedMapping(t *testing.T) {
	mapping := schema.NewMetricsSchema()

	mapping = mapping.AddStringMapping("Region")
	mapping = mapping.AddStringMapping("Host")

	_, err := json.Marshal(mapping)
	require.NoError(t, err)

	require.Equal(t, len(mapping), 6)
	require.Equal(t, "Timestamp", mapping[0].Column)
	require.Equal(t, "datetime", mapping[0].DataType)
	require.Equal(t, "0", mapping[0].Properties.Ordinal)

	require.Equal(t, "SeriesId", mapping[1].Column)
	require.Equal(t, "long", mapping[1].DataType)
	require.Equal(t, "1", mapping[1].Properties.Ordinal)

	require.Equal(t, "Labels", mapping[2].Column)
	require.Equal(t, "dynamic", mapping[2].DataType)
	require.Equal(t, "2", mapping[2].Properties.Ordinal)

	require.Equal(t, "Value", mapping[3].Column)
	require.Equal(t, "real", mapping[3].DataType)
	require.Equal(t, "3", mapping[3].Properties.Ordinal)

	require.Equal(t, "Region", mapping[4].Column)
	require.Equal(t, "string", mapping[4].DataType)
	require.Equal(t, "4", mapping[4].Properties.Ordinal)

	require.Equal(t, "Host", mapping[5].Column)
	require.Equal(t, "string", mapping[5].DataType)
	require.Equal(t, "5", mapping[5].Properties.Ordinal)
}

func TestNormalizeAdxIdentifier(t *testing.T) {
	test := func(t *testing.T, expected string, input string) {
		t.Helper()
		require.Equal(t, expected, schema.NormalizeAdxIdentifier(input))
	}

	// Kusto does not appear to allow characters outside of the ASCII range
	test(t, "Redis", "Redis⌘")
	test(t, "Redis", "Redis⌘日本語")
	test(t, "Redis", "_Re-d.is")
	test(t, "9Redis", "9Redis")
	test(t, "RedisRequests", "$Redis::/Requests")
	// Invalid UTF-8 character
	test(t, "RedisRequests", "Redis\xc3\x28Requests")

	// max length
	test(t, strings.Repeat("a", 1024), strings.Repeat("a", 1025))
	// preserve first portion
	test(t, fmt.Sprintf("b%s", strings.Repeat("a", 1023)), fmt.Sprintf("b%s", strings.Repeat("a", 1025)))
	// Remove invalid characters in middle, still truncate
	test(t, fmt.Sprintf("aaaa%s", strings.Repeat("a", 1020)), fmt.Sprintf("aaaa_%s", strings.Repeat("a", 1025)))
}

func BenchmarkNormalizeAdxIdentifier(b *testing.B) {
	b.Run("No changes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			schema.NormalizeAdxIdentifier("Redis")
		}
	})

	b.Run("Normalize characters out", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			schema.NormalizeAdxIdentifier("Redis_Metrics")
		}
	})
}

func TestAppendNormalizeAdxIdentifier(t *testing.T) {
	test := func(t *testing.T, expected string, input string) {
		t.Helper()
		require.Equal(t, expected, string(schema.AppendNormalizeAdxIdentifier([]byte{}, []byte(input))))
	}

	// Kusto does not appear to allow characters outside of the ASCII range
	test(t, "Redis", "Redis⌘")
	test(t, "Redis", "Redis⌘日本語")
	test(t, "Redis", "_Re-d.is")
	test(t, "9Redis", "9Redis")
	test(t, "RedisRequests", "$Redis::/Requests")
	// Invalid UTF-8 character
	test(t, "RedisRequests", "Redis\xc3\x28Requests")

	// max length
	test(t, strings.Repeat("a", 1024), strings.Repeat("a", 1025))
	// preserve first portion
	test(t, fmt.Sprintf("b%s", strings.Repeat("a", 1023)), fmt.Sprintf("b%s", strings.Repeat("a", 1025)))
	// Remove invalid characters in middle, still truncate
	test(t, fmt.Sprintf("aaaa%s", strings.Repeat("a", 1020)), fmt.Sprintf("aaaa_%s", strings.Repeat("a", 1025)))
}

func TestNormalizeMetricName(t *testing.T) {
	require.Equal(t, "Redis", string(schema.NormalizeMetricName([]byte("__redis__"))))
	require.Equal(t, "UsedCpuUserChildren", string(schema.NormalizeMetricName([]byte("used_cpu_user_children"))))
	require.Equal(t, "Host1", string(schema.NormalizeMetricName([]byte("host_1"))))
	require.Equal(t, "Region", string(schema.NormalizeMetricName([]byte("region"))))
	require.Equal(t, "Region1", string(schema.NormalizeMetricName([]byte("region_⌘1"))))
	require.Equal(t, "Region", string(schema.NormalizeMetricName([]byte("region-._"))))
	require.Equal(t, "JobEtcdRequestLatency75pctlrate5m", string(schema.NormalizeMetricName([]byte("Job:etcdRequestLatency:75pctlrate5m"))))
	require.Equal(t, "TestLimit", string(schema.NormalizeMetricName([]byte("Test$limit"))))
	require.Equal(t, "TestRateLimit", string(schema.NormalizeMetricName([]byte("Test::Rate$limit"))))
	// Invalid UTF-8 character
	require.Equal(t, "RedisRequests", schema.NormalizeAdxIdentifier("Redis\xc3\x28Requests"))

	// max length
	require.Equal(t, fmt.Sprintf("A%s", strings.Repeat("a", 1023)), string(schema.NormalizeMetricName([]byte(strings.Repeat("a", 1025)))))
	// preserve first portion
	require.Equal(t, fmt.Sprintf("B%s", strings.Repeat("a", 1023)), string(schema.NormalizeMetricName([]byte(fmt.Sprintf("b%s", strings.Repeat("a", 1025))))))
	// Remove invalid characters in middle, still truncate
	require.Equal(t, fmt.Sprintf("AaaaA%s", strings.Repeat("a", 1019)), string(schema.NormalizeMetricName([]byte(fmt.Sprintf("aaaa_%s", strings.Repeat("a", 1025))))))
}

func TestAppendCSVHeaderWithValidMapping(t *testing.T) {
	mapping := schema.NewMetricsSchema()
	expected := "Timestamp:datetime,SeriesId:long,Labels:dynamic,Value:real\n"
	result := schema.AppendCSVHeader(nil, mapping)
	require.Equal(t, expected, string(result))
}

func TestAppendCSVHeaderWithEmptyMapping(t *testing.T) {
	var mapping schema.SchemaMapping
	expected := "\n"
	result := schema.AppendCSVHeader(nil, mapping)
	require.Equal(t, expected, string(result))
}

func TestAppendCSVHeaderWithCustomMapping(t *testing.T) {
	mapping := schema.SchemaMapping{
		{Column: "CustomColumn1", DataType: "string"},
		{Column: "CustomColumn2", DataType: "int"},
	}
	expected := "CustomColumn1:string,CustomColumn2:int\n"
	result := schema.AppendCSVHeader(nil, mapping)
	require.Equal(t, expected, string(result))
}

func TestUnmarshalSchemaWithValidData(t *testing.T) {
	data := "Timestamp:datetime,SeriesId:long,Labels:dynamic,Value:real\n"
	expected := schema.SchemaMapping{
		{Column: "Timestamp", DataType: "datetime", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "0"}},
		{Column: "SeriesId", DataType: "long", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "1"}},
		{Column: "Labels", DataType: "dynamic", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "2"}},
		{Column: "Value", DataType: "real", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "3"}},
	}
	result, err := schema.UnmarshalSchema(data)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestUnmarshalSchemaWithEmptyData(t *testing.T) {
	data := ""
	expected := schema.SchemaMapping{}
	result, err := schema.UnmarshalSchema(data)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestUnmarshalSchemaWithInvalidData(t *testing.T) {
	data := "Timestamp:datetime,SeriesId:long,Labels\n"
	_, err := schema.UnmarshalSchema(data)
	require.Error(t, err)
}

func TestUnmarshalSchemaWithExtraNewline(t *testing.T) {
	data := "Timestamp:datetime,SeriesId:long,Labels:dynamic,Value:real\n\n"
	expected := schema.SchemaMapping{
		{Column: "Timestamp", DataType: "datetime", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "0"}},
		{Column: "SeriesId", DataType: "long", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "1"}},
		{Column: "Labels", DataType: "dynamic", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "2"}},
		{Column: "Value", DataType: "real", Properties: struct {
			Ordinal    string `json:"Ordinal,omitempty"`
			ConstValue string `json:"ConstValue,omitempty"`
		}{Ordinal: "3"}},
	}
	result, err := schema.UnmarshalSchema(data)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestUnmarshalSchemaWithMissingDataType(t *testing.T) {
	data := "Timestamp:datetime,SeriesId:long,Labels:dynamic,Value\n"
	_, err := schema.UnmarshalSchema(data)
	require.Error(t, err)
}
