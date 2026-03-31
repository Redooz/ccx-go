package fileread

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/anton-abyzov/ccx-go/internal/tool"
)

// Known image extensions.
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".webp": true, ".svg": true, ".ico": true,
	".tiff": true, ".tif": true, ".avif": true, ".heic": true,
}

// Known binary extensions (non-image).
var binaryExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".zst": true,
	".7z": true, ".rar": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".class": true, ".o": true, ".a": true, ".pyc": true, ".pyo": true,
	".wasm": true, ".bin": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mkv": true, ".wav": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
}

type input struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// Tool implements the FileRead tool.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "Read" }
func (t *Tool) Description() string { return "Read a file from the local filesystem." }

func (t *Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string", "description": "Absolute path to the file"},
			"offset": {"type": "integer", "description": "Line number to start reading from (1-based, default 1)"},
			"limit": {"type": "integer", "description": "Maximum number of lines to read (default 2000)"}
		},
		"required": ["file_path"]
	}`)
}

func (t *Tool) Execute(ctx context.Context, rawInput json.RawMessage) (*tool.Result, error) {
	var in input
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return &tool.Result{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	if in.FilePath == "" {
		return &tool.Result{Content: "file_path is required", IsError: true}, nil
	}

	// Check file exists and get info
	info, err := os.Stat(in.FilePath)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error: %v", err), IsError: true}, nil
	}
	if info.IsDir() {
		return &tool.Result{Content: "error: path is a directory, not a file. Use Bash with ls to list directory contents.", IsError: true}, nil
	}

	ext := strings.ToLower(filepath.Ext(in.FilePath))

	// Detect image files
	if imageExts[ext] {
		return &tool.Result{
			Content: fmt.Sprintf("[Image file: %s (%d bytes)]\nThis is a %s image file. Use an image viewer to display it.", in.FilePath, info.Size(), strings.TrimPrefix(ext, ".")),
		}, nil
	}

	// Detect binary files by extension
	if binaryExts[ext] {
		return &tool.Result{
			Content: fmt.Sprintf("[Binary file: %s (%d bytes)]\nThis is a binary file and cannot be displayed as text.", in.FilePath, info.Size()),
		}, nil
	}

	// Open file and check for binary content via sniffing
	f, err := os.Open(in.FilePath)
	if err != nil {
		return &tool.Result{Content: fmt.Sprintf("error opening file: %v", err), IsError: true}, nil
	}
	defer f.Close()

	// Sniff first 512 bytes for binary content
	sniffBuf := make([]byte, 512)
	n, _ := f.Read(sniffBuf)
	if n > 0 {
		sniffBuf = sniffBuf[:n]
		contentType := http.DetectContentType(sniffBuf)
		if isBinaryContentType(contentType) && !isTextLike(sniffBuf) {
			return &tool.Result{
				Content: fmt.Sprintf("[Binary file: %s (%d bytes, detected: %s)]\nThis file appears to be binary and cannot be displayed as text.", in.FilePath, info.Size(), contentType),
			}, nil
		}
	}

	// Seek back to start for reading
	if _, err := f.Seek(0, 0); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error seeking file: %v", err), IsError: true}, nil
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 2000
	}

	offset := in.Offset
	if offset <= 0 {
		offset = 1 // 1-based, default to start
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	var b strings.Builder
	lineNum := 0
	linesRead := 0

	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		lineNum++
		if lineNum < offset {
			continue
		}
		if linesRead >= limit {
			break
		}
		fmt.Fprintf(&b, "%d\t%s\n", lineNum, scanner.Text())
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return &tool.Result{Content: fmt.Sprintf("error reading file: %v", err), IsError: true}, nil
	}

	content := b.String()
	if content == "" {
		if lineNum == 0 {
			content = "(empty file)"
		} else {
			content = fmt.Sprintf("(no lines in range: file has %d lines, requested offset %d)", lineNum, offset)
		}
	}

	return &tool.Result{Content: content}, nil
}

// isBinaryContentType checks if the detected content type is binary.
func isBinaryContentType(ct string) bool {
	return !strings.HasPrefix(ct, "text/") && ct != "application/json" && ct != "application/xml" && ct != "application/javascript"
}

// isTextLike checks if the buffer content looks like text (no null bytes).
func isTextLike(buf []byte) bool {
	for _, b := range buf {
		if b == 0 {
			return false
		}
	}
	return true
}
