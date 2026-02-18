package chezmoi

import (
	"reflect"
	"testing"
)

func TestEntryFilterArgsBuildsLongFlags(t *testing.T) {
	t.Parallel()

	got := entryFilterArgs(EntryFilter{
		Include: []EntryType{EntryFiles, EntryTemplates},
		Exclude: []EntryType{EntryDirs, EntryScripts},
	})

	want := []string{
		"--include=files",
		"--include=templates",
		"--exclude=dirs",
		"--exclude=scripts",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entryFilterArgs() = %#v, want %#v", got, want)
	}
}

func TestEntryFilterArgsEmptyFilter(t *testing.T) {
	t.Parallel()

	got := entryFilterArgs(EntryFilter{})
	if len(got) != 0 {
		t.Fatalf("entryFilterArgs() = %#v, want empty slice", got)
	}
}
