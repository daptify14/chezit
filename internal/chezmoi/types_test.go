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
		"--include=files,templates",
		"--exclude=dirs,scripts",
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

func TestGitSyncStateString(t *testing.T) {
	t.Parallel()

	cases := map[GitSyncState]string{
		GitSyncUnknown:    "unknown",
		GitSyncNoUpstream: "no-upstream",
		GitSyncSynced:     "synced",
		GitSyncAhead:      "ahead",
		GitSyncBehind:     "behind",
		GitSyncDiverged:   "diverged",
		GitSyncState(99):  "unknown",
	}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("GitSyncState(%d).String() = %q, want %q", int(state), got, want)
		}
	}
}
