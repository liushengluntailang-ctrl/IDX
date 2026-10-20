FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# Go言語とPythonを追加
RUN apk update && apk add --no-cache go python3

WORKDIR /app

# C# の実行用プロジェクトを事前に作っておく（初回ビルド高速化）
RUN mkdir -p /app/csharp && cd /app/csharp && dotnet new console

# ソースコードをコピーしてGoサーバーをコンパイル
COPY main.go .
COPY public/ ./public/
RUN go build -o server main.go

EXPOSE 10000

CMD ["./server"]
