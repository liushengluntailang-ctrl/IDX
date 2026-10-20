package main

import (
	"encoding/json"
	"fmt"
	"io"
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

type CmdRequest struct {
	Cmd string `json:"cmd"`
}

type DeleteRequest struct {
	Path string `json:"path"`
}

func main() {
	workspaceRoot := "/app/workspace"

	// 初期サンプルSVGファイルを作成
	svgSample := filepath.Join(workspaceRoot, "logo.svg")
	if _, err := os.Stat(svgSample); os.IsNotExist(err) {
		os.WriteFile(svgSample, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#007acc;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#4ec9b0;stop-opacity:1" />
    </linearGradient>
  </defs>
  <circle cx="100" cy="100" r="80" fill="url(#grad)" />
  <path d="M70 60 L140 100 L70 140 Z" fill="#ffffff" />
</svg>`), 0644)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	})

	// ファイル一覧
	http.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		var list []FileItem
		filepath.Walk(workspaceRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil || p == workspaceRoot {
				return nil
			}
			rel, _ := filepath.Rel(workspaceRoot, p)
			list = append(list, FileItem{Name: info.Name(), Path: rel, IsDir: info.IsDir()})
			return nil
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	// テキスト読み込み
	http.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(workspaceRoot, r.URL.Query().Get("path"))
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
	})

	// バイナリ配信 (MP3, MP4, 画像などの直接ストリーミング)
	http.HandleFunc("/api/raw", func(w http.ResponseWriter, r *http.Request) {
		fullPath := filepath.Join(workspaceRoot, r.URL.Query().Get("path"))
		http.ServeFile(w, r, fullPath)
	})

	// 保存
	http.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		var req SaveRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		os.WriteFile(fullPath, []byte(req.Content), 0644)
		w.Write([]byte("OK"))
	})

	// 作成
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

	// ★ 1. ファイル / フォルダ削除 API
	http.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		var req DeleteRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		err := os.RemoveAll(fullPath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("OK"))
	})

	// ★ 2. 対話型ターミナル実行 API (任意のLinuxコマンド)
	http.HandleFunc("/api/cmd", func(w http.ResponseWriter, r *http.Request) {
		var req CmdRequest
		json.NewDecoder(r.Body).Decode(&req)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %s && %s", workspaceRoot, req.Cmd))
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			fmt.Fprintf(w, "Command error: %v", err)
			return
		}
		w.Write(out)
	})

	// プロジェクトビルド & 実行 API
	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req ExecRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		dir := filepath.Dir(fullPath)
		ext := strings.ToLower(filepath.Ext(fullPath))

		var cmd *exec.Cmd

		switch ext {
		case ".cs":
			cmd = exec.Command("sh", "-c", fmt.Sprintf(`
				cd '%s'
				if [ -f *.csproj ]; then
					dotnet run
				else
					cp *.cs /app/csharp_env/ 2>/dev/null || true
					cd /app/csharp_env && dotnet run
				fi
			`, dir))
		case ".rs":
			cmd = exec.Command("sh", "-c", fmt.Sprintf(`
				cd '%s'
				if [ -f Cargo.toml ]; then
					cargo run
				else
					rustc *.rs -o /tmp/rust_app && /tmp/rust_app
				fi
			`, dir))
		case ".go":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && go run .", dir))
		case ".cpp", ".c":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && g++ *.cpp *.c -o /tmp/native_app 2>/dev/null || g++ '%s' -o /tmp/native_app && /tmp/native_app", dir, fullPath))
		case ".py":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && python3 '%s'", dir, fullPath))
		case ".js", ".ts":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && node '%s'", dir, fullPath))
		case ".sh":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && sh '%s'", dir, fullPath))
		default:
			fmt.Fprintf(w, "実行対象外のファイルです: %s", ext)
			return
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(w, "%s\n[ビルド/実行エラー]", string(out))
			return
		}
		w.Write(out)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	fmt.Println("Custom Project IDX Server is running on :" + port)
	http.ListenAndServe(":"+port, nil)
}
