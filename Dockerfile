# 1. ビルド済みの公式code-serverからバイナリだけを拝借する
FROM codercom/code-server:latest AS code-server-src

# 2. メインのAlpine + .NET 8 SDK環境
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# 多言語ツールと、code-server実行に必要な最小限のライブラリだけを高速インストール
RUN apk update && apk add --no-cache \
    nodejs npm git bash curl build-base python3 go rust cargo \
    libstdc++ gcompat

# 公式のcode-serverバイナリを丸ごとAlpineに移植
COPY --from=code-server-src /usr/lib/code-server /usr/lib/code-server
RUN ln -s /usr/lib/code-server/bin/code-server /usr/local/bin/code-server

# Render用ポート 10000
EXPOSE 10000

# 起動コマンド
CMD ["code-server", "--bind-addr", "0.0.0.0:10000", "--auth", "none", "/root"]
