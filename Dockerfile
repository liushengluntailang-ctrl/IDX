# Microsoft公式 Alpine Linux + .NET 8 SDK
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# Kerberosのビルドに必要な krb5-dev と多言語ツールを一括インストール
RUN apk update && apk add --no-cache \
    nodejs npm git bash curl build-base python3 go rust cargo \
    krb5-dev libsecret-dev

# code-server のインストール (krb5-devが入ったのでkerberosのコンパイルが100%通る)
RUN npm install -g code-server

# Render用ポート 10000
EXPOSE 10000

# 起動コマンド
CMD ["code-server", "--bind-addr", "0.0.0.0:10000", "--auth", "none", "/root"]
