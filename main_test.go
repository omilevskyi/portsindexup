package main

import (
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
