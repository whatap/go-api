// Package whatapas provides helper functions for Aerospike client instrumentation.
package whatapas

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	aerospike "github.com/aerospike/aerospike-client-go/v6"
	whatapsql "github.com/whatap/go-api/sql"
)

// Max number of bin names/values to collect (matches SQL_PARAM_MAX_COUNT)
const maxBinCount = 20

// GetDbhost extracts the database host information from an Aerospike client.
// Returns "aerospike" as fallback if host information cannot be determined.
// This function is safe to call with nil client or when no nodes are available.
func GetDbhost(client *aerospike.Client) string {
	if client == nil {
		return "aerospike"
	}
	nodes := client.GetNodes()
	if nodes == nil || len(nodes) == 0 {
		return "aerospike"
	}
	h := nodes[0].GetHost()
	if h == nil {
		return "aerospike"
	}
	return fmt.Sprintf("aerospike://%s:%d", h.Name, h.Port)
}

// FormatSQL formats the SQL-like string for Aerospike operations.
// Format: "METHOD namespace.set.userKey [bin1,bin2,...]"
// Example: "Put test.demo.key1 [name,age,email]"
func FormatSQL(method string, key *aerospike.Key, binNames []string) string {
	if key == nil {
		return method
	}

	keyStr := fmt.Sprintf("%s.%s.%v", key.Namespace(), key.SetName(), key.Value())

	if len(binNames) == 0 {
		return fmt.Sprintf("%s %s", method, keyStr)
	}

	// Limit bin names
	if len(binNames) > maxBinCount {
		binNames = binNames[:maxBinCount]
	}

	return fmt.Sprintf("%s %s [%s]", method, keyStr, strings.Join(binNames, ","))
}

// ExtractBinNames extracts bin names from BinMap.
// Returns sorted slice for consistent output.
func ExtractBinNames(bins aerospike.BinMap) []string {
	if bins == nil {
		return nil
	}

	names := make([]string, 0, len(bins))
	for name := range bins {
		names = append(names, name)
	}
	sort.Strings(names) // Sort for consistent ordering
	return names
}

// ExtractBinValues extracts bin values from BinMap in sorted key order.
// Values are returned as []interface{} for use with StartWithParamArray.
func ExtractBinValues(bins aerospike.BinMap) []interface{} {
	if bins == nil {
		return nil
	}

	names := ExtractBinNames(bins)
	values := make([]interface{}, 0, len(names))
	for _, name := range names {
		values = append(values, bins[name])
	}
	return values
}

// ExtractBinNamesFromSlice extracts bin names from []*Bin slice.
func ExtractBinNamesFromSlice(bins []*aerospike.Bin) []string {
	if bins == nil {
		return nil
	}

	names := make([]string, 0, len(bins))
	for _, bin := range bins {
		if bin != nil {
			names = append(names, bin.Name)
		}
	}
	return names
}

// ExtractBinValuesFromSlice extracts bin values from []*Bin slice.
func ExtractBinValuesFromSlice(bins []*aerospike.Bin) []interface{} {
	if bins == nil {
		return nil
	}

	values := make([]interface{}, 0, len(bins))
	for _, bin := range bins {
		if bin != nil {
			values = append(values, bin.Value)
		}
	}
	return values
}

// WrapPut wraps client.Put with key/bins information collection.
// This function collects SQL-like query info: "Put namespace.set.key [bin1,bin2]"
func WrapPut(ctx context.Context, client *aerospike.Client, key *aerospike.Key, bins aerospike.BinMap, fn func() error) error {
	dbhost := GetDbhost(client)
	binNames := ExtractBinNames(bins)
	sqlStr := FormatSQL("Put", key, binNames)
	values := ExtractBinValues(bins)

	sqlCtx, _ := whatapsql.StartWithParamArray(ctx, dbhost, sqlStr, values)
	err := fn()
	whatapsql.End(sqlCtx, err)
	return err
}

// WrapPutBins wraps client.Put with []*Bin (variadic) arguments.
func WrapPutBins(ctx context.Context, client *aerospike.Client, key *aerospike.Key, bins []*aerospike.Bin, fn func() error) error {
	dbhost := GetDbhost(client)
	binNames := ExtractBinNamesFromSlice(bins)
	sqlStr := FormatSQL("Put", key, binNames)
	values := ExtractBinValuesFromSlice(bins)

	sqlCtx, _ := whatapsql.StartWithParamArray(ctx, dbhost, sqlStr, values)
	err := fn()
	whatapsql.End(sqlCtx, err)
	return err
}

// WrapGet wraps client.Get with key information collection.
// binNames are the optional bin names to retrieve.
func WrapGet(ctx context.Context, client *aerospike.Client, key *aerospike.Key, binNames []string, fn func() (*aerospike.Record, error)) (*aerospike.Record, error) {
	dbhost := GetDbhost(client)
	sqlStr := FormatSQL("Get", key, binNames)

	sqlCtx, _ := whatapsql.Start(ctx, dbhost, sqlStr)
	record, err := fn()
	whatapsql.End(sqlCtx, err)
	return record, err
}

// WrapDelete wraps client.Delete with key information collection.
func WrapDelete(ctx context.Context, client *aerospike.Client, key *aerospike.Key, fn func() (bool, error)) (bool, error) {
	dbhost := GetDbhost(client)
	sqlStr := FormatSQL("Delete", key, nil)

	sqlCtx, _ := whatapsql.Start(ctx, dbhost, sqlStr)
	existed, err := fn()
	whatapsql.End(sqlCtx, err)
	return existed, err
}

// WrapExists wraps client.Exists with key information collection.
func WrapExists(ctx context.Context, client *aerospike.Client, key *aerospike.Key, fn func() (bool, error)) (bool, error) {
	dbhost := GetDbhost(client)
	sqlStr := FormatSQL("Exists", key, nil)

	sqlCtx, _ := whatapsql.Start(ctx, dbhost, sqlStr)
	exists, err := fn()
	whatapsql.End(sqlCtx, err)
	return exists, err
}

// WrapGeneric wraps any Aerospike operation with basic method name.
// Use this for methods where key/bins extraction is not applicable.
// The result parameter should be a pointer to store the result.
func WrapGeneric(ctx context.Context, client *aerospike.Client, method string, fn func() error) error {
	dbhost := GetDbhost(client)
	sqlCtx, _ := whatapsql.Start(ctx, dbhost, method)
	err := fn()
	whatapsql.End(sqlCtx, err)
	return err
}

// WrapGenericWithKey wraps any Aerospike operation with key information.
func WrapGenericWithKey(ctx context.Context, client *aerospike.Client, method string, key *aerospike.Key, fn func() error) error {
	dbhost := GetDbhost(client)
	sqlStr := FormatSQL(method, key, nil)
	sqlCtx, _ := whatapsql.Start(ctx, dbhost, sqlStr)
	err := fn()
	whatapsql.End(sqlCtx, err)
	return err
}

// extractBinsFromInterface extracts bin names and values from interface{}.
// Supports: aerospike.BinMap, []*aerospike.Bin, *aerospike.Bin
func extractBinsFromInterface(bins interface{}) ([]string, []interface{}) {
	if bins == nil {
		return nil, nil
	}

	switch b := bins.(type) {
	case aerospike.BinMap:
		return ExtractBinNames(b), ExtractBinValues(b)
	case []*aerospike.Bin:
		return ExtractBinNamesFromSlice(b), ExtractBinValuesFromSlice(b)
	case *aerospike.Bin:
		if b != nil {
			return []string{b.Name}, []interface{}{b.Value}
		}
	default:
		// Try reflection for unknown types
		v := reflect.ValueOf(bins)
		if v.Kind() == reflect.Slice {
			names := make([]string, 0, v.Len())
			values := make([]interface{}, 0, v.Len())
			for i := 0; i < v.Len(); i++ {
				elem := v.Index(i).Interface()
				if bin, ok := elem.(*aerospike.Bin); ok && bin != nil {
					names = append(names, bin.Name)
					values = append(values, bin.Value)
				}
			}
			return names, values
		}
	}
	return nil, nil
}
