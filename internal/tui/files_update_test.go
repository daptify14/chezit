package tui

import "testing"

func filesTreeRowIndex(t *testing.T, m Model, absPath string) int {
	t.Helper()
	for i, row := range m.activeTreeRows() {
		if !row.node.isDir && row.node.absPath == absPath {
			return i
		}
	}
	t.Fatalf("no tree row for %q", absPath)
	return -1
}

func TestManagedReloadKeepsSelectedFileInTree(t *testing.T) {
	files := []string{"/home/test/.bashrc", "/home/test/.config/nvim/init.lua", "/home/test/.zshrc"}
	m := newTestModel(WithManagedFiles(files))
	m.filesTab.cursor = filesTreeRowIndex(t, m, "/home/test/.zshrc")

	reloaded := append([]string{"/home/test/.aliases"}, files...)
	m, _ = sendMsg(t, m, chezmoiManagedLoadedMsg{files: reloaded, gen: m.gen})

	if got := m.selectedManagedPath(); got != "/home/test/.zshrc" {
		t.Fatalf("selected path after reload = %q, want /home/test/.zshrc", got)
	}
	if want := filesTreeRowIndex(t, m, "/home/test/.zshrc"); m.filesTab.cursor != want {
		t.Fatalf("cursor after reload = %d, want %d", m.filesTab.cursor, want)
	}
}

func TestManagedReloadKeepsSelectedFileInFlatView(t *testing.T) {
	files := []string{"/home/test/.bashrc", "/home/test/.zshrc"}
	m := newTestModel(WithManagedFiles(files))
	m.filesTab.treeView = false
	m.filesTab.cursor = 1

	reloaded := append([]string{"/home/test/.aliases"}, files...)
	m, _ = sendMsg(t, m, chezmoiManagedLoadedMsg{files: reloaded, gen: m.gen})

	if m.filesTab.cursor != 2 {
		t.Fatalf("cursor after reload = %d, want 2", m.filesTab.cursor)
	}
	if got := m.selectedManagedPath(); got != "/home/test/.zshrc" {
		t.Fatalf("selected path after reload = %q, want /home/test/.zshrc", got)
	}
}

func TestManagedReloadClampsCursorWhenSelectedFileRemoved(t *testing.T) {
	files := []string{"/home/test/.bashrc", "/home/test/.profile", "/home/test/.zshrc"}
	m := newTestModel(WithManagedFiles(files))
	m.filesTab.treeView = false
	m.filesTab.cursor = 2

	m, _ = sendMsg(t, m, chezmoiManagedLoadedMsg{files: files[:2], gen: m.gen})

	if m.filesTab.cursor != 1 {
		t.Fatalf("cursor after reload = %d, want 1", m.filesTab.cursor)
	}
}

func TestManagedFirstLoadStartsAtTop(t *testing.T) {
	m := newTestModel()

	m, _ = sendMsg(t, m, chezmoiManagedLoadedMsg{files: []string{"/home/test/.bashrc", "/home/test/.zshrc"}, gen: m.gen})

	if m.filesTab.cursor != 0 {
		t.Fatalf("cursor after first load = %d, want 0", m.filesTab.cursor)
	}
}
