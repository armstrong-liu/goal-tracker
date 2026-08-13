#!/usr/bin/env bash
# build.sh —— 多平台编译 Goal Tracker
# 用法： ./scripts/build.sh [version]
# 输出： dist/gt-<version>-<os>-<arch>[.exe]

set -e

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
DIST_DIR="dist"
PACKAGE="goal-tracker"
MAIN_PACKAGE="."

mkdir -p "$DIST_DIR"

# 定义目标平台
# 格式: "GOOS/GOARCH"
TARGETS=(
  "windows/amd64"
  "windows/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

echo "🏗️  编译 $PACKAGE $VERSION"
echo "   输出目录: $DIST_DIR"
echo ""

# 注入版本号
LDFLAGS="-X 'goal-tracker/internal/cmd.Version=$VERSION'"

SUCCESS=0
FAIL=0

for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"

  # 确定输出文件名
  output="$DIST_DIR/gt-${VERSION}-${goos}-${goarch}"
  if [ "$goos" = "windows" ]; then
    output="${output}.exe"
  fi

  printf "   🔨 %-20s ... " "$goos/$goarch"

  if GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
      -ldflags "$LDFLAGS" \
      -o "$output" \
      "$MAIN_PACKAGE" 2>/dev/null; then
    size=$(du -h "$output" | cut -f1)
    printf "✅ %s\n" "$size"
    SUCCESS=$((SUCCESS + 1))
  else
    printf "❌ 失败\n"
    FAIL=$((FAIL + 1))
  fi
done

echo ""
echo "📊 编译结果: 成功 $SUCCESS / 失败 $FAIL"
echo "📦 产物列表:"
ls -lh "$DIST_DIR"/gt-* 2>/dev/null || echo "   （无）"
