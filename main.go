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

	// 初期Qt QMLファイル
	qmlSample := filepath.Join(workspaceRoot, "main.qml")
	if _, err := os.Stat(qmlSample); os.IsNotExist(err) {
		os.WriteFile(qmlSample, []byte(`import QtQuick 2.15
import QtQuick.Controls 2.15

ApplicationWindow {
    visible: true
    width: 320
    height: 480
    title: "Qt Quick Window"

    property int count: 0

    Column {
        anchors.centerIn: parent
        spacing: 16

        Text {
            text: "Qt QML Interactive"
            font.pixelSize: 20
            color: "#00b4d8"
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: "Count: " + count
            font.pixelSize: 18
            color: "#ffffff"
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Button {
            text: "Qt Button (Click Me)"
            anchors.horizontalCenter: parent.horizontalCenter
            onClicked: {
                count += 1
            }
        }
    }
}`), 0644)
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

	// 読み込み
	http.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(workspaceRoot, r.URL.Query().Get("path"))
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
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

	// ★ 単一ファイルではなく「プロジェクト単位」で爆速実行するエンジン
	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req ExecRequest
		json.NewDecoder(r.Body).Decode(&req)
		fullPath := filepath.Join(workspaceRoot, req.Path)
		dir := filepath.Dir(fullPath)
		ext := strings.ToLower(filepath.Ext(fullPath))

		var cmd *exec.Cmd

		switch ext {
		case ".cs":
			// プロジェクトルートを探して dotnet run、なければディレクトリ内の全 .cs をプロジェクト化して実行
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
			// Cargo.toml があれば cargo run、なければディレクトリ内の全 .rs を結合コンパイル
			cmd = exec.Command("sh", "-c", fmt.Sprintf(`
				cd '%s'
				if [ -f Cargo.toml ]; then
					cargo run
				else
					rustc *.rs -o /tmp/rust_app && /tmp/rust_app
				fi
			`, dir))

		case ".go":
			// ディレクトリ内の全 Go ファイル（パッケージ全体）をリンクして実行
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && go run .", dir))

		case ".cpp", ".c":
			// ディレクトリ内の全 C/C++ ファイルをリンクビルド
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && g++ *.cpp *.c -o /tmp/native_app 2>/dev/null || g++ '%s' -o /tmp/native_app && /tmp/native_app", dir, fullPath))

		case ".py":
			// ディレクトリ全体を import できるようにカレントディレクトリから実行
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && python3 '%s'", dir, fullPath))

		case ".js", ".ts":
			cmd = exec.Command("sh", "-c", fmt.Sprintf("cd '%s' && node '%s'", dir, fullPath))

		default:
			fmt.Fprintf(w, "実行対象外のファイルです: %s", ext)
			return
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(w, "%s\n[ビルド/実行エラー: exit status]", string(out))
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
