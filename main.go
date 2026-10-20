package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FileItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

type SaveRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type CreateRequest struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

type ExecRequest struct {
	Path string `json:"path"`
}

func main() {
	workspaceRoot := "/app/workspace"

	// 初期ファイルがなければ作成
	initFile := filepath.Join(workspaceRoot, "Program.cs")
	if _, err := os.Stat(initFile); os.IsNotExist(err) {
		os.WriteFile(initFile, []byte("using System;\n\nclass Program {\n    static void Main() {\n        Console.WriteLine(\"Hello from Custom Alpine IDX!\");\n    }\n}"), 0644)
	}

	// 1. トップ画面 (index.html)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	})

	// 2. ファイル一覧取得 API
	http.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		var list []FileItem
		filepath.Walk(workspaceRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil || p == workspaceRoot {
				return nil
			}
			rel, _ := filepath.Rel(workspaceRoot, p)
			list = append(list, FileItem{
				Name:  info.Name(),
				Path:  rel,
				IsDir: info.IsDir(),
			})
			return nil
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	// 3. ファイル内容読み込み API
	http.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(workspaceRoot, r.URL.Query().Get("path"))
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
	})

	// 4. ファイル保存 API
	http.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		var req SaveRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		os.WriteFile(fullPath, []byte(req.Content), 0644)
		w.Write([]byte("OK"))
	})

	// 5. 新規ファイル / フォルダ作成 API
	http.HandleFunc("/api/create", func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		if req.IsDir {
			os.MkdirAll(fullPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(""), 0644)
		}
		w.Write([]byte("OK"))
	})

	// 6. 爆速マルチ言語 実行 API (拡張子判定)
	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req ExecRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		dir := filepath.Dir(fullPath)
		ext := strings.ToLower(filepath.Ext(fullPath))

		var cmd *exec.Cmd

		switch ext {
		case ".cs": // C#
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cp '%s' /app/csharp_env/Program.cs && cd /app/csharp_env && dotnet run", fullPath))
		case ".py": // Python
			cmd = exec.Command("python3", fullPath)
		case ".go": // Go
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && go run '%s'", dir, fullPath))
		case ".rs": // Rust
			cmd = exec.Command("sh", "-c", fmt.Sprintf("rustc '%s' -o /tmp/rust_app && /tmp/rust_app", fullPath))
		case ".cpp", ".cc": // C++
			cmd = exec.Command("sh", "-c", fmt.Sprintf("g++ '%s' -o /tmp/cpp_app && /tmp/cpp_app", fullPath))
		case ".c": // C言語
			cmd = exec.Command("sh", "-c", fmt.Sprintf("gcc '%s' -o /tmp/c_app && /tmp/c_app", fullPath))
		case ".js": // JavaScript
			cmd = exec.Command("node", fullPath)
		case ".sh": // Shell
			cmd = exec.Command("sh", fullPath)
		default:
			fmt.Fprintf(w, "未対応の拡張子です: %s", ext)
			return
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(w, "%s\n[Process exited with error]", string(out))
			return
		}
		w.Write(out)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	fmt.Println("Custom IDX Server is listening on port " + port)
	http.ListenAndServe(":"+port, nil)
}
