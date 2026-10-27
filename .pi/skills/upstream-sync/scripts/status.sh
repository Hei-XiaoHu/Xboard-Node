#!/usr/bin/env bash
# 汇总 Xboard-Node 及 4 个内核 fork 相对上游 / cedar 的更新情况。
#
#   status.sh           拉取所有远程并打印报告（只读，不改任何分支）
#   status.sh --mark    把当前各上游 ref 记为“已审阅”，写入 ../state.env
#   status.sh --no-fetch  不 fetch，直接用本地已有的远程 ref
#
# 环境变量 PROXY_DIR 可覆盖仓库所在目录（默认：本仓库的上一级）。
set -uo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="$SKILL_DIR/state.env"
REPO_ROOT="$(git -C "$SKILL_DIR" rev-parse --show-toplevel)"
PROXY_DIR="${PROXY_DIR:-$(dirname "$REPO_ROOT")}"
XN="$PROXY_DIR/Xboard-Node" XR="$PROXY_DIR/Xray-core" SB="$PROXY_DIR/sing-box"
SQ="$PROXY_DIR/sing-quic" RE="$PROXY_DIR/REALITY"

MARK=0 FETCH=1
for a in "$@"; do
  case "$a" in
    --mark) MARK=1 ;;
    --no-fetch) FETCH=0 ;;
    *) echo "unknown arg: $a" >&2; exit 2 ;;
  esac
done

for d in "$XN" "$XR" "$SB" "$SQ" "$RE"; do
  [ -d "$d/.git" ] || { echo "缺少仓库: $d" >&2; exit 1; }
done

if [ "$FETCH" = 1 ]; then
  echo "fetching ..." >&2
  git -C "$XN" fetch -q upstream --tags & git -C "$XN" fetch -q origin &
  git -C "$XR" fetch -q upstream --tags & git -C "$XR" fetch -q cedar & git -C "$XR" fetch -q origin &
  git -C "$SB" fetch -q upstream --tags & git -C "$SB" fetch -q cedar & git -C "$SB" fetch -q origin &
  git -C "$SQ" fetch -q upstream & git -C "$SQ" fetch -q origin &
  git -C "$RE" fetch -q upstream & git -C "$RE" fetch -q origin &
  wait
fi

# 读取已审阅状态
declare -A SEEN=()
if [ -f "$STATE" ]; then
  while IFS='=' read -r k v; do [ -n "$k" ] && [ "${k:0:1}" != "#" ] && SEEN[$k]=$v; done < "$STATE"
fi
declare -A NOW=()

sha() { git -C "$1" rev-parse --verify -q "$2^{commit}"; }
# go.mod 里某模块的版本 -> 可用于 git 的 rev（伪版本取末尾 hash，否则就是 tag）
gomod_rev() {
  local v; v=$(awk -v m="$2" '$1==m && $2 ~ /^v/ {print $2; exit}' "$1")
  [ -z "$v" ] && v=$(grep -E "^[[:space:]]*$2 v" "$1" | head -1 | awk '{print $2}')
  case "$v" in *-*-[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) echo "${v##*-}" ;; *) echo "$v" ;; esac
}
latest_tag() { # $1 repo, $2 stable|any
  if [ "$2" = stable ]; then
    git -C "$1" -c versionsort.suffix=- tag -l 'v*' --sort=-v:refname | grep -vE -- '-(alpha|beta|rc)' | head -1
  else
    git -C "$1" -c versionsort.suffix=- tag -l 'v*' --sort=-v:refname | head -1
  fi
}
# 打印 ref 相对上次审阅的新提交
since_seen() { # $1 key $2 repo $3 ref $4 label
  local key=$1 repo=$2 ref=$3 label=$4 cur old n
  cur=$(sha "$repo" "$ref") || { echo "  [$label] ref $ref 不存在"; return; }
  NOW[$key]=$cur
  old=${SEEN[$key]:-}
  if [ -z "$old" ]; then
    echo "  [$label] $ref @ ${cur:0:12}（从未标记审阅，显示最近 5 条）"
    git -C "$repo" log --no-merges --format='      %h %ad %s' --date=short -5 "$ref"
  elif [ "$old" = "$cur" ]; then
    echo "  [$label] $ref 自上次审阅无变化"
  else
    n=$(git -C "$repo" rev-list --count "$old..$ref" 2>/dev/null || echo "?")
    echo "  [$label] $ref 自上次审阅 (${old:0:12}) 新增 $n 个提交："
    git -C "$repo" log --no-merges --format='      %h %ad %s' --date=short "$old..$ref" | head -40
  fi
}
merged() { git -C "$1" merge-base --is-ancestor "$2" "${3:-main}" 2>/dev/null; }
local_state() { # 工作区/推送状态
  local s; s=$(git -C "$1" status --porcelain | wc -l)
  local ab; ab=$(git -C "$1" rev-list --left-right --count main...origin/main 2>/dev/null | awk '{print "未推送 "$1" / 落后 origin "$2}')
  echo "  本地: 分支 $(git -C "$1" branch --show-current)，未提交改动 $s 个文件，$ab"
}

echo
echo "================ Xboard-Node (upstream: cedar2025/Xboard-Node dev) ================"
local_state "$XN"
n=$(git -C "$XN" rev-list --count main..upstream/dev)
echo "  upstream/dev 中尚未合入 main 的提交: $n"
[ "$n" != 0 ] && git -C "$XN" log --no-merges --format='      %h %ad %s' --date=short main..upstream/dev | head -40
since_seen XN_UPSTREAM "$XN" upstream/dev "upstream/dev"
echo "  上游 go.mod 使用的内核（看原作者是否换了 cedar 补丁版本）:"
git -C "$XN" show upstream/dev:go.mod | grep -E '^replace .*(sing-box|xray-core|sing-quic|reality)' | sed 's/^/      /'
echo "  我们 go.mod 的 replace:"
grep -E '^replace .*Hei-XiaoHu' "$XN/go.mod" | sed 's/^/      /'

echo
echo "================ Xray-core (upstream: XTLS/Xray-core 发布 tag) ================"
local_state "$XR"
cur=$(git -C "$XR" describe --tags --abbrev=0 --match 'v*' main 2>/dev/null)
st=$(latest_tag "$XR" stable); any=$(latest_tag "$XR" any)
echo "  main 已合入: $cur    上游最新正式版: $st    最新(含预发布): $any"
if [ -n "$st" ] && ! merged "$XR" "$st"; then
  echo "  >>> 有新正式版未合入：$cur..$st 共 $(git -C "$XR" rev-list --count "$cur..$st") 个提交"
  git -C "$XR" log --no-merges --format='      %h %ad %s' --date=short "$cur..$st" | head -40
fi
since_seen XR_CEDAR "$XR" cedar/main "cedar/main"
xr_reality=$(gomod_rev "$XR/go.mod" github.com/xtls/reality)
echo "  Xray main 的 go.mod 用 REALITY: $xr_reality"
if [ -n "$st" ] && ! merged "$XR" "$st"; then
  up_reality=$(git -C "$XR" show "$st:go.mod" | awk '$1=="github.com/xtls/reality"{print $2}')
  echo "  上游 $st 的 go.mod 用 REALITY: $up_reality"
fi

echo
echo "================ sing-box (upstream: SagerNet/sing-box 发布 tag) ================"
local_state "$SB"
cur=$(git -C "$SB" describe --tags --abbrev=0 --match 'v*' main 2>/dev/null)
st=$(latest_tag "$SB" stable); any=$(latest_tag "$SB" any)
echo "  main 已合入: $cur    上游最新正式版: $st    最新(含预发布): $any"
if [ -n "$st" ] && ! merged "$SB" "$st"; then
  echo "  >>> 有新正式版未合入：$cur..$st 共 $(git -C "$SB" rev-list --count "$cur..$st") 个提交"
  git -C "$SB" log --no-merges --format='      %h %ad %s' --date=short "$cur..$st" | head -40
fi
since_seen SB_CEDAR_TESTING "$SB" cedar/testing "cedar/testing"
echo "  cedar 最近更新的分支（原作者补丁一般在 testing，留意是否出现新分支）:"
git -C "$SB" for-each-ref --sort=-committerdate refs/remotes/cedar --count=5 \
  --format='      %(refname:short) %(committerdate:short) %(subject)' | grep -v '^      cedar ' | head -4
sb_quic=$(gomod_rev "$SB/go.mod" github.com/sagernet/sing-quic)
echo "  sing-box main 的 go.mod require sing-quic: $sb_quic"

echo
echo "================ sing-quic (目标：sing-box go.mod 里 require 的提交) ================"
local_state "$SQ"
if merged "$SQ" "$sb_quic"; then
  echo "  已包含 sing-box 需要的 $sb_quic ✓"
else
  echo "  >>> 尚未合入 sing-box 需要的 $sb_quic"
fi
if [ -n "${st:-}" ] && ! merged "$SB" "$st"; then
  up_quic=$(git -C "$SB" show "$st:go.mod" | awk '$1=="github.com/sagernet/sing-quic"{print $2}')
  echo "  上游 sing-box $st 需要 sing-quic: $up_quic"
fi
n=$(git -C "$SQ" rev-list --count main..upstream/main)
echo "  upstream/main 领先 main $n 个提交（仅供参考，一般跟随 sing-box 的版本）"
since_seen SQ_UPSTREAM "$SQ" upstream/main "upstream/main"

echo
echo "================ REALITY (目标：Xray go.mod 里的版本) ================"
local_state "$RE"
if merged "$RE" "$xr_reality"; then
  echo "  已包含 Xray 需要的 $xr_reality ✓"
else
  echo "  >>> 尚未合入 Xray 需要的 $xr_reality"
fi
n=$(git -C "$RE" rev-list --count main..upstream/main)
echo "  upstream/main 领先 main $n 个提交"
since_seen RE_UPSTREAM "$RE" upstream/main "upstream/main"

echo
echo "================ Xboard-Node go.mod replace 是否指向各 fork 的 main ================"
for pair in "github.com/xtls/xray-core:$XR" "github.com/sagernet/sing-box:$SB" "github.com/sagernet/sing-quic:$SQ" "github.com/xtls/reality:$RE"; do
  mod=${pair%%:*} dir=${pair#*:}
  v=$(awk -v m="$mod" '$1=="replace" && $2==m {print $5; exit}' "$XN/go.mod")
  h=${v##*-}
  head=$(git -C "$dir" rev-parse --short=12 origin/main)
  if [ "$h" = "$head" ]; then echo "  $mod -> $h ✓ (= origin/main)"; else echo "  $mod -> $h ≠ origin/main $head"; fi
done

if [ "$MARK" = 1 ]; then
  {
    echo "# upstream-sync 已审阅位置（status.sh --mark 自动生成，$(date -u +%FT%TZ)）"
    for k in $(printf '%s\n' "${!NOW[@]}" | sort); do echo "$k=${NOW[$k]}"; done
  } > "$STATE"
  echo
  echo "已写入审阅标记: $STATE"
fi
