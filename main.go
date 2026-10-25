package main

import (
	"bytes"
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

func main() {
	workspaceRoot := "/app/workspace"

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

	// ファイル読込
	http.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(workspaceRoot, r.URL.Query().Get("path"))
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
	})

	// ファイル書込 / 保存
	http.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path, Content string }
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(req.Content), 0644)
		w.Write([]byte("OK"))
	})

	// ファイル作成
	http.HandleFunc("/api/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path  string `json:"path"`
			IsDir bool   `json:"isDir"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, strings.TrimSpace(req.Path))
		if req.IsDir {
			os.MkdirAll(fullPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(""), 0644)
		}
		w.Write([]byte("OK"))
	})

	// 削除
	http.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path string }
		json.NewDecoder(r.Body).Decode(&req)
		os.RemoveAll(filepath.Join(workspaceRoot, req.Path))
		w.Write([]byte("OK"))
	})

	// ターミナルコマンド実行
	http.HandleFunc("/api/cmd", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Cmd string }
		json.NewDecoder(r.Body).Decode(&req)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %s && %s", workspaceRoot, req.Cmd))
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}
		w.Write(out)
	})

	// AIエージェントAPI中継 (OpenRouterキーをサーバー内環境変数から使用)
	http.HandleFunc("/api/ai", func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			http.Error(w, "RenderのEnvironmentにOPENROUTER_API_KEYを設定してください", 400)
			return
		}
		body, _ := io.ReadAll(r.Body)
		aiReq, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(body))
		aiReq.Header.Set("Authorization", "Bearer "+apiKey)
		aiReq.Header.Set("Content-Type", "application/json")
		aiReq.Header.Set("HTTP-Referer", "https://github.com/liushengluntailang-ctrl/IDX")

		resp, err := (&http.Client{}).Do(aiReq)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	fmt.Println("Server running on :" + port)
	http.ListenAndServe(":"+port, nil)
}
