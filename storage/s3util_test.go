package storage

import "testing"

func TestApplyS3Prefix(t *testing.T) {
	cases := []struct {
		prefix, name, want string
	}{
		{"", "foo", "foo"},
		{"prefix/", "bar", "prefix/bar"},
		{"prefix/", "", "prefix/"},
		{"", "", ""},
		// New cases: prefix without trailing slash
		{"prefix", "bar", "prefix/bar"},
		{"prefix", "", "prefix/"},
		{"prefix", "baz/qux", "prefix/baz/qux"},
	}
	for _, c := range cases {
		got := applyS3Prefix(c.prefix, c.name)
		if got != c.want {
			t.Errorf("applyS3Prefix(%q, %q) = %q, want %q", c.prefix, c.name, got, c.want)
		}
	}
}
