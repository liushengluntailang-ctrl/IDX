FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# 主要言語のツールチェーンを一括インストール
# C/C++ (build-base), Python, Go, Rust, Node.js
RUN apk update && apk add --no-cache \
    go python3 nodejs npm rust cargo build-base git bash

WORKDIR /app

# 作業用ディレクトリとC#用事前プロジェクトの準備
RUN mkdir -p /app/workspace /app/csharp_env && \
    cd /app/csharp_env && dotnet new console

# ソースコードとUIをコピーしてサーバーをビルド
COPY main.go .
COPY index.html .
RUN go build -o server main.go

EXPOSE 10000

CMD ["./server"]
