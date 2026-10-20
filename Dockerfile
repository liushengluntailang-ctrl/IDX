# Microsoft公式の Alpine Linux + .NET 8 SDK
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# 多言語ツールチェーンを一括インストール
RUN apk update && apk add --no-cache \
    nodejs npm git bash curl build-base python3 go rust cargo

# code-server (ブラウザVS Code) の導入
RUN npm install -g code-server --unsafe-perm

# Render用ポート 10000
EXPOSE 10000

# 起動（作業ディレクトリを /root に設定）
CMD ["code-server", "--bind-addr", "0.0.0.0:10000", "--auth", "none", "/root"]
