#!/usr/bin/env bash
set -euo pipefail

# 可配置项：
# - FEISHU_REPO: 目标仓库，格式 owner/repo（默认 riba2534/feishu-cli）
# - FEISHU_BINARY_NAME: 二进制名（默认 feishu-cli）
# - FEISHU_INSTALL_DIR: 安装目录（默认 /usr/local/bin）
REPO="${FEISHU_REPO:-${GITHUB_REPOSITORY:-riba2534/feishu-cli}}"
BINARY_NAME="${FEISHU_BINARY_NAME:-feishu-cli}"
DEFAULT_INSTALL_DIR="${FEISHU_INSTALL_DIR:-/usr/local/bin}"
DEFAULT_INSTALL_VERSION="${FEISHU_VERSION:-}"

# 颜色输出
info()  { printf "\033[34m[INFO]\033[0m  %s\n" "$*" >&2; }
ok()    { printf "\033[32m[OK]\033[0m    %s\n" "$*" >&2; }
err()   { printf "\033[31m[ERROR]\033[0m %s\n" "$*" >&2; }

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *)       err "不支持的操作系统: $(uname -s)"; exit 1 ;;
    esac
}

# 检测架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64|armv8*) echo "arm64" ;;
        *)             err "不支持的架构: $(uname -m)"; exit 1 ;;
    esac
}

http_get() {
    local url="$1"
    if command -v curl &>/dev/null; then
        curl -fsSL --max-time 30 "$url"
    elif command -v wget &>/dev/null; then
        wget -qO- "$url"
    else
        err "需要安装 curl 或 wget"; exit 1
    fi
}

get_version_from_release_json() {
    local json="$1"
    if command -v jq &>/dev/null; then
        echo "$json" | jq -r '.tag_name // empty'
    else
        echo "$json" | grep -oE '"tag_name":\s*"[^"]+"' | sed -n '1p' | sed 's/.*"tag_name":\s*"\([^"]*\)".*/\1/'
    fi
}

# 获取版本
get_latest_version() {
    if [ -n "$DEFAULT_INSTALL_VERSION" ]; then
        echo "$DEFAULT_INSTALL_VERSION"
        return
    fi

    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local json version
    json=$(http_get "$url")
    version=$(get_version_from_release_json "$json")
    if [ -z "$version" ]; then
        err "无法获取最新版本号"; exit 1
    fi
    echo "$version"
}

# 识别可下载资产名（兼容常见短横线/下划线写法）
asset_candidates() {
    local version="$1" os="$2" arch="$3"
    printf '%s_%s_%s-%s.tar.gz\n' "$BINARY_NAME" "$version" "$os" "$arch"
    printf '%s_%s_%s_%s.tar.gz\n' "$BINARY_NAME" "$version" "$os" "$arch"
    if [ "$os" = "windows" ]; then
        printf '%s_%s_%s_%s.tar.gz\n' "$BINARY_NAME" "$version" "$os" "$arch"
        printf '%s_%s_%s-%s.tar.gz\n' "$BINARY_NAME" "$version" "$os" "$arch"
    fi
}

detect_install_dir() {
    # 1. 已安装时更新到同一目录
    local existing
    existing=$(command -v "$BINARY_NAME" 2>/dev/null || true)
    if [ -n "$existing" ]; then
        local real_path
        real_path=$(readlink -f "$existing" 2>/dev/null || echo "$existing")
        echo "$(dirname "$real_path")"
        return
    fi

    # 2. GOBIN
    if [ -n "${GOBIN:-}" ] && [ -d "$GOBIN" ]; then
        echo "$GOBIN"
        return
    fi

    # 3. GOPATH/bin
    local gopath_bin
    if [ -n "${GOPATH:-}" ]; then
        gopath_bin="${GOPATH}/bin"
    elif command -v go &>/dev/null; then
        gopath_bin="$(go env GOPATH 2>/dev/null)/bin"
    fi
    if [ -n "${gopath_bin:-}" ] && [ -d "$gopath_bin" ]; then
        echo "$gopath_bin"
        return
    fi

    # 4. 默认
    echo "$DEFAULT_INSTALL_DIR"
}

download_from_candidates() {
    local tmpdir="$1" version="$2" os="$3" arch="$4"
    local asset_name download_url
    local url_prefix="https://github.com/${REPO}/releases/download/${version}"

    while IFS= read -r asset_name; do
        [ -z "$asset_name" ] && continue
        download_url="${url_prefix}/${asset_name}"
        info "尝试下载 ${download_url}"
        if command -v curl &>/dev/null; then
            if curl -fSL --progress-bar -o "${tmpdir}/${asset_name}" "$download_url"; then
                echo "$asset_name"
                return 0
            fi
        else
            if wget -q --show-progress -O "${tmpdir}/${asset_name}" "$download_url"; then
                echo "$asset_name"
                return 0
            fi
        fi
    done < <(asset_candidates "$version" "$os" "$arch")

    err "未找到可下载的资产。请确认以下任一命名存在："
    asset_candidates "$version" "$os" "$arch" | sed 's/^/  - /'
    return 1
}

# 下载并安装
install() {
    local os arch version install_dir tmpdir asset_name download_file binary_path target_name
    local repo_url

    os=$(detect_os)
    arch=$(detect_arch)
    version=$(get_latest_version)
    install_dir=$(detect_install_dir)

    info "仓库: ${REPO}"
    info "检测到平台: ${os}/${arch}"
    info "版本: ${version}"
    info "安装目录: ${install_dir}"

    # 检查是否已安装相同版本
    if command -v "$BINARY_NAME" &>/dev/null; then
        local current
        current=$("$BINARY_NAME" --version 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        if [ "$current" = "$version" ]; then
            ok "已是最新版本 ${version}，无需更新"
            exit 0
        fi
        info "当前版本: ${current}，将更新到 ${version}"
    fi

    # 创建临时目录
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    asset_name=$(download_from_candidates "$tmpdir" "$version" "$os" "$arch")
    download_file="${tmpdir}/${asset_name}"
    repo_url="https://github.com/${REPO}/releases/download/${version}/${asset_name}"

    info "解压安装包 ${repo_url}"
    tar -xzf "$download_file" -C "$tmpdir"

    # 查找可执行文件
    if [ "$os" = "windows" ]; then
        target_name="${BINARY_NAME}.exe"
        installed_name="${BINARY_NAME}.exe"
    else
        target_name="$BINARY_NAME"
        installed_name="$BINARY_NAME"
    fi
    binary_path=$(find "$tmpdir" -type f -name "$target_name" | head -n 1)
    if [ -z "$binary_path" ]; then
        err "解压后未找到 ${target_name} 二进制文件"; exit 1
    fi
    chmod +x "$binary_path"

    # 安装到目标目录
    info "安装到 ${install_dir}/${installed_name}"
    if [ -w "$install_dir" ]; then
        mv "$binary_path" "${install_dir}/${installed_name}"
    else
        sudo mv "$binary_path" "${install_dir}/${installed_name}"
    fi

    # 验证安装
    if command -v "$installed_name" &>/dev/null; then
        ok "安装成功: $("$installed_name" --version 2>/dev/null || true)"
    else
        ok "已安装到 ${install_dir}/${installed_name}"
        echo "  如果命令未找到，请确认 ${install_dir} 在 PATH 中"
    fi
}

install
