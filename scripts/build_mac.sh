#!/usr/bin/env bash
# 打包 macOS 版本：wails build + 把 bin/ 拷贝进 .app 包内
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "▶ wails build (production, darwin/arm64)"
wails build -clean

APP="$ROOT/build/bin/Flux.app"
if [ ! -d "$APP" ]; then
  echo "❌ 构建失败：找不到 $APP"
  exit 1
fi

echo "▶ 复制 bin/ → $APP/Contents/Resources/bin/"
cp -R "$ROOT/bin" "$APP/Contents/Resources/bin"

SIZE=$(du -sh "$APP" | cut -f1)
echo
echo "✅ 完成：$APP  ($SIZE)"
echo "▶ 启动：open \"$APP\""
