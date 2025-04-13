package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestStrip(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"noDash", "nodash", "nodash"},
		{"dashFirst", "-first", "-"},
		{"dashLast", "last-", "last-"},
		{"dashMiddle", "middle-dash", "middle-"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := strip(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("strip() = \"%+v\", want \"%+v\"", got, tt.want)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		name, source, search, replace, want string
	}{
		{"000", "", "", "", ""},
		{"001", "", "", "replace", "replace"},
		{"010", "", "search", "", ""},
		{"011", "", "search", "replace", ""},
		{"100", "source", "", "", "source"},
		{"101", "source", "", "replace", "replacesource"},
		{"110a", "source", "search", "", "source"},
		{"110b", "source", "sou", "", "rce"},
		{"110c", "source", "rce", "", "sou"},
		{"110d", "source", "ur", "", "soce"},
		{"111a", "source", "search", "replace", "source"},
		{"111b", "source", "sou", "replace", "replacerce"},
		{"111c", "source", "rce", "replace", "soureplace"},
		{"111d", "source", "ur", "replace", "soreplacece"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := replace(tt.source, tt.search, tt.replace)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("replace() = \"%+v\", want \"%+v\"", got, tt.want)
			}
		})
	}
}

func TestRootDirectory(t *testing.T) {
	cases := []struct {
		name, want string
	}{
		{"root", "/"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := rootDirectory()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("replace() = \"%+v\", want \"%+v\"", got, tt.want)
			}
		})
	}
}

func TestUpdatePath(t *testing.T) {
	cases := []struct {
		dst    []string
		src    []string
		idx    int
		prefix string
		count  int
		want   []string
	}{
		{
			dst:    []string{"", "", ""},
			src:    []string{"a/b/c", "d/e/f", "g/h/i"},
			idx:    1,
			prefix: "prefix",
			count:  2,
			want:   []string{"", "prefix/e/f", ""},
		},
		{
			dst:    []string{"", "", ""},
			src:    []string{"a/b/c", "d/e/f", "g/h/i"},
			idx:    0,
			prefix: "prefix",
			count:  3,
			want:   []string{"prefix/a/b/c", "", ""},
		},
		{
			dst:    []string{"", "", ""},
			src:    []string{"a/b/c", "", "g/h/i"},
			idx:    2,
			prefix: "prefix",
			count:  1,
			want:   []string{"", "", "prefix/i"},
		},
		{
			dst:    []string{"", "", ""},
			src:    []string{"a/b/c", "d/e/f", "g/h/i"},
			idx:    3,
			prefix: "prefix",
			count:  2,
			want:   []string{"", "", ""},
		},
		{
			dst:    []string{"", "", ""},
			src:    []string{"a/b/c", "d/e/f", "g/h/i"},
			idx:    1,
			prefix: "prefix",
			count:  4,
			want:   []string{"", "", ""},
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			updatePath(tt.dst, tt.src, tt.idx, tt.prefix, tt.count)
			for i := range tt.dst {
				if tt.dst[i] != tt.want[i] {
					t.Errorf("updatePath() = %v, want %v", tt.dst, tt.want)
				}
			}
		})
	}
}

func TestSafeUpdate(t *testing.T) {
	cases := []struct {
		dst  []string
		didx int
		src  []string
		sidx int
		want []string
	}{
		{
			dst:  []string{"", "", ""},
			didx: 1,
			src:  []string{"a", "b", "c"},
			sidx: 1,
			want: []string{"", "b", ""},
		},
		{
			dst:  []string{"", "", ""},
			didx: 0,
			src:  []string{"a", "b", "c"},
			sidx: 2,
			want: []string{"c", "", ""},
		},
		{
			dst:  []string{"", "", ""},
			didx: 2,
			src:  []string{"a", "", "c"},
			sidx: 2,
			want: []string{"", "", "c"},
		},
		{
			dst:  []string{"x", "y", "z"},
			didx: 1,
			src:  []string{"a", "b", "c"},
			sidx: 3,
			want: []string{"x", "y", "z"},
		},
		{
			dst:  []string{"x", "y", "z"},
			didx: 3,
			src:  []string{"a", "b", "c"},
			sidx: 1,
			want: []string{"x", "y", "z"},
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			safeUpdate(tt.dst, tt.didx, tt.src, tt.sidx)
			for i := range tt.dst {
				if tt.dst[i] != tt.want[i] {
					t.Errorf("safeUpdate() = %v, want %v", tt.dst, tt.want)
				}
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}

func TestUpdateDependency(t *testing.T) {
	cases := []struct {
		pstr         *string
		replacements map[string]string
		from         string
		to           string
		want         string
	}{
		{
			pstr:         ptr("a b c"),
			replacements: map[string]string{"a": "x", "b": "y"},
			from:         "c",
			to:           "z",
			want:         "x y z",
		},
		{
			pstr:         ptr("d e f"),
			replacements: map[string]string{"d": "u", "e": "v"},
			from:         "f",
			to:           "w",
			want:         "u v w",
		},
		{
			pstr:         ptr("g h i"),
			replacements: map[string]string{"g": "m", "h": "n"},
			from:         "i",
			to:           "o",
			want:         "m n o",
		},
		{
			pstr:         ptr("j k l"),
			replacements: map[string]string{"j": "p", "k": "q"},
			from:         "l",
			to:           "r",
			want:         "p q r",
		},
		{
			pstr:         ptr(""),
			replacements: map[string]string{"a": "x", "b": "y"},
			from:         "c",
			to:           "z",
			want:         "",
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			updateDependency(tt.pstr, tt.replacements, tt.from, tt.to)
			if *tt.pstr != tt.want {
				t.Errorf("updateDependency() = %v, want %v", *tt.pstr, tt.want)
			}
		})
	}
}
