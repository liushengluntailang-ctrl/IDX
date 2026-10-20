# 1. 公式code-serverから資材を拝借
FROM codercom/code-server:latest AS code-server-src

# 2. Alpine + .NET 8 SDK
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine

# 多言語ツールとAlpine純正のNode.js
RUN apk update && apk add --no-cache \
    nodejs npm git bash curl build-base python3 go rust cargo \
    libstdc++ gcompat

# code-serverの資材をコピー
COPY --from=code-server-src /usr/lib/code-server /usr/lib/code-server

# 【ここが解決の鍵】同梱のDebian用nodeを削除し、Alpine純正のNode.jsにリンクを張り替える
RUN rm -f /usr/lib/code-server/lib/node && \
    ln -s /usr/bin/node /usr/lib/code-server/lib/node && \
    ln -s /usr/lib/code-server/bin/code-server /usr/local/bin/code-server

# Render用ポート 10000
EXPOSE 10000

# 起動コマンド
CMD ["code-server", "--bind-addr", "0.0.0.0:10000", "--auth", "none", "/root"]
