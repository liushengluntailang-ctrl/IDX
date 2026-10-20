package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

type ExecuteRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

func main() {
	// 直下の index.html などの静的ファイルを配信
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	})

	// コード実行API
	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var req ExecuteRequest
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		var cmd *exec.Cmd

		if req.Language == "csharp" {
			// C# の実行
			os.WriteFile("/tmp/Program.cs", []byte(req.Code), 0644)
			cmd = exec.Command("sh", "-c", "cp /tmp/Program.cs /app/csharp/Program.cs && cd /app/csharp && dotnet run")
		} else if req.Language == "python" {
			// Python の実行
			os.WriteFile("/tmp/script.py", []byte(req.Code), 0644)
			cmd = exec.Command("python3", "/tmp/script.py")
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(w, "実行エラー:\n%s\n%s", string(out), err.Error())
			return
		}
		w.Write(out)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}
