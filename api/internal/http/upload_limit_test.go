package http

import "testing"

func TestAnOverSizedUploadIsRefusedInASizeAPersonReads(t *testing.T) {
	for _, row := range []struct {
		bytes int64
		want  string
	}{
		{bytes: 33554432, want: "32 MB"},
		{bytes: 1048576, want: "1 MB"},
		{bytes: 1572864, want: "1.5 MB"},
		{bytes: 5000, want: "4.9 KB"},
		{bytes: 512, want: "512 bytes"},
		{bytes: 2147483648, want: "2 GB"},
	} {
		if got := readableSize(row.bytes); got != row.want {
			t.Errorf("readableSize(%d) = %q, want %q", row.bytes, got, row.want)
		}
	}
}
