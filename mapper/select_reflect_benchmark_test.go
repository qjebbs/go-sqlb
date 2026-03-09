package mapper

import (
	"testing"
	"time"
)

func BenchmarkSelectReflectScan(b *testing.B) {
	// make sure the definition of userListItem is the same as in BenchmarkSelectModelScan
	type Model struct {
		ID      int        `sqlb:"col:id"`
		Created *time.Time `sqlb:"col:created"`
		Updated *time.Time `sqlb:"col:updated"`
		Deleted *time.Time `sqlb:"col:deleted"`
	}

	type User struct {
		Model `sqlb:"table:users"`
		OrgID int    `sqlb:"col:org_id"`
		Name  string `sqlb:"col:name"`
	}

	type Org struct {
		Model `sqlb:"table:orgs"`
		Name  string `sqlb:"col:name"`
	}
	type userListItem struct {
		User
		Org
	}
	dest := &userListItem{}
	info, err := getStructInfo(dest)
	if err != nil {
		b.Fatal(err)
	}
	dests := make([]fieldInfo, 0)
	for _, col := range info.columns {
		dests = append(dests, col)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prepareScanDestinations(dest, dests)
	}
}
