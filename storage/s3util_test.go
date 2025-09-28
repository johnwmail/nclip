package storage

import "testing"

func TestNormalizeS3Prefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"/", ""},
		{"prefix", "prefix/"},
		{"/prefix/", "prefix/"},
		{"prefix/", "prefix/"},
		{"/prefix", "prefix/"},
	}
	for _, c := range cases {
		got := normalizeS3Prefix(c.in)
		if got != c.want {
			t.Errorf("normalizeS3Prefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestApplyS3Prefix(t *testing.T) {
	cases := []struct {
		prefix, name, want string
	}{
		{"", "foo", "foo"},
		{"prefix/", "bar", "prefix/bar"},
		{"prefix/", "", "prefix/"},
		{"", "", ""},
	}
	for _, c := range cases {
		got := applyS3Prefix(c.prefix, c.name)
		if got != c.want {
			t.Errorf("applyS3Prefix(%q, %q) = %q, want %q", c.prefix, c.name, got, c.want)
		}
	}
}
