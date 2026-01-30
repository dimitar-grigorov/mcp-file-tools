//go:build ignore

// Manual test for all MCP server operations.
// Run with: go run test_server.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver"
	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

var failed = 0

func check(name string, ok bool) {
	fmt.Printf("%-40s ", name)
	if ok {
		fmt.Println("OK")
	} else {
		fmt.Println("FAIL")
		failed++
	}
}

func main() {
	tempDir, _ := os.MkdirTemp("", "mcp-test-*")
	defer os.RemoveAll(tempDir)

	h := handler.NewHandler([]string{tempDir})
	ctx := context.Background()

	fmt.Printf("Server version: %s\n\n", filetoolsserver.Version)

	// Basic tools
	r1, o1, _ := h.HandleListAllowedDirectories(ctx, nil, handler.ListAllowedDirectoriesInput{})
	check("list_allowed_directories", !r1.IsError && len(o1.Directories) == 1)

	r2, o2, _ := h.HandleListEncodings(ctx, nil, handler.ListEncodingsInput{})
	check("list_encodings", !r2.IsError && len(o2.Encodings) > 0)

	// Write/Read UTF-8
	testFile := filepath.Join(tempDir, "test.txt")
	r3, _, _ := h.HandleWriteFile(ctx, nil, handler.WriteFileInput{Path: testFile, Content: "Hello!", Encoding: "utf-8"})
	check("write_file (UTF-8)", !r3.IsError)

	r4, o4, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: testFile})
	check("read_text_file (UTF-8)", !r4.IsError && o4.Content == "Hello!")

	// Write/Read CP1251
	cyrillicFile := filepath.Join(tempDir, "cyrillic.txt")
	r5, _, _ := h.HandleWriteFile(ctx, nil, handler.WriteFileInput{Path: cyrillicFile, Content: "Здравей!", Encoding: "cp1251"})
	check("write_file (CP1251)", !r5.IsError)

	r6, o6, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: cyrillicFile, Encoding: "cp1251"})
	check("read_text_file (CP1251)", !r6.IsError && o6.Content == "Здравей!")

	// Detection and info
	r7, o7, _ := h.HandleDetectEncoding(ctx, nil, handler.DetectEncodingInput{Path: testFile})
	check("detect_encoding", !r7.IsError && o7.Encoding != "")

	r8, o8, _ := h.HandleGetFileInfo(ctx, nil, handler.GetFileInfoInput{Path: testFile})
	check("get_file_info", !r8.IsError && o8.IsFile && o8.Size > 0)

	// Directory operations
	r9, o9, _ := h.HandleListDirectory(ctx, nil, handler.ListDirectoryInput{Path: tempDir})
	check("list_directory", !r9.IsError && len(o9.Files) >= 2)

	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "subdir", "nested.txt"), []byte("x"), 0644)

	r10, o10, _ := h.HandleDirectoryTree(ctx, nil, handler.DirectoryTreeInput{Path: tempDir})
	var tree []handler.TreeEntry
	json.Unmarshal([]byte(o10.Tree), &tree)
	check("directory_tree", !r10.IsError && len(tree) >= 2)

	r11, o11, _ := h.HandleDirectoryTree(ctx, nil, handler.DirectoryTreeInput{Path: tempDir, ExcludePatterns: []string{"*.txt"}})
	json.Unmarshal([]byte(o11.Tree), &tree)
	hasFile := false
	for _, e := range tree {
		if e.Type == "file" {
			hasFile = true
		}
	}
	check("directory_tree (exclude)", !r11.IsError && !hasFile)

	// Head/tail
	multiFile := filepath.Join(tempDir, "multi.txt")
	os.WriteFile(multiFile, []byte("a\nb\nc\nd\n"), 0644)
	head, tail := 2, 2
	r12, o12, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: multiFile, Head: &head})
	check("read_text_file (head)", !r12.IsError && o12.Content == "a\nb")

	r13, o13, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: multiFile, Tail: &tail})
	check("read_text_file (tail)", !r13.IsError && o13.Content == "d\n")

	// Auto-detect and encoding registry
	r14, o14, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: cyrillicFile})
	check("read_text_file (auto-detect)", !r14.IsError && o14.DetectedEncoding != "")

	enc, ok := encoding.Get("cp1251")
	check("encoding registry", ok && enc != nil)

	// Security: path validation
	r16, _, _ := h.HandleReadTextFile(ctx, nil, handler.ReadTextFileInput{Path: filepath.Join(tempDir, "..", "..", "etc", "passwd")})
	check("path validation (deny)", r16.IsError)

	fmt.Println()
	if failed > 0 {
		fmt.Printf("FAILED: %d test(s)\n", failed)
		os.Exit(1)
	}
	fmt.Println("All tests passed!")
}
