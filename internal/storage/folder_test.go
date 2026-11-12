package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListFolderShowsMarkdownWithoutFollowingLinks(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"a.md": "a", "B.MARKDOWN": "b", ".hidden.md": "hidden", "other.txt": "text", "notes/nested.md": "nested",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.Symlink(root, filepath.Join(root, "loop"))
	folder, err := ListFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(folder.Children) != 3 || folder.Children[0].Name != "notes" ||
		folder.Children[1].Name != "a.md" || folder.Children[2].Name != "B.MARKDOWN" {
		t.Fatalf("children: %#v", folder.Children)
	}
	if len(folder.Children[0].Children) != 1 || folder.Children[0].Children[0].Name != "nested.md" {
		t.Fatalf("nested: %#v", folder.Children[0].Children)
	}
}
