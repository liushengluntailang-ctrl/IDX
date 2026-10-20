FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# Go言語とPythonを追加
RUN apk update && apk add --no-cache go python3

WORKDIR /app

# C# の実行用プロジェクトを事前に作っておく（実行を爆速にするため）
RUN mkdir -p /app/csharp && cd /app/csharp && dotnet new console

# ソースコードとHTMLを直下からコピー
COPY main.go .
COPY index.html .

# Goサーバーをビルド
RUN go build -o server main.go

EXPOSE 10000

CMD ["./server"]
