package api

import (
	"reflect"
	"strings"
	"testing"
)

func TestTransactionSourceIDValidation(t *testing.T) {
	ids, err := parseTransactionSourceIDs("3, 2,3")
	if err != nil || !reflect.DeepEqual(ids, []int64{3, 2}) {
		t.Fatalf("ids %+v: %v", ids, err)
	}
	for _, raw := range []string{"", " ", "-1", "0", "1.2", "1,,2", "1;DROP TABLE transactions", "999999999999999999999", strings.Repeat("1,", 1000) + "2"} {
		if _, err := parseTransactionSourceIDs(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
