#!/usr/bin/env bash
# 汇总 Xboard-Node 及 4 个内核 fork 相对上游的更新情况。
#
# 主要上游（活跃，优先看）：XTLS/Xray-core、SagerNet/sing-box、SagerNet/sing-quic、XTLS/REALITY、
#   cedar2025/Xboard-Node（节点本体的原作者）、cedar2025/Xboard（面板，节点 API 的另一端）。
# 参考来源（更新少）：cedar2025/Xray-core、cedar2025/sing-box —— 原作者给内核打的 Xboard 补丁。
#
#   status.sh             拉取所有远程并打印报告（只读，不改任何分支）
#   status.sh --mark      把当前各上游位置记为“已审阅”，写入 ../state.env
#   status.sh --no-fetch  不 fetch，直接用本地已有的远程 ref（面板仍会联网查询）
#
# 环境变量 PROXY_DIR 可覆盖仓库所在目录（默认：本仓库的上一级）。
set -uo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="$SKILL_DIR/state.env"
REPO_ROOT="$(git -C "$SKILL_DIR" rev-parse --show-toplevel)"
PROXY_DIR="${PROXY_DIR:-$(dirname "$REPO_ROOT")}"
XN="$PROXY_DIR/Xboard-Node" XR="$PROXY_DIR/Xray-core" SB="$PROXY_DIR/sing-box"
SQ="$PROXY_DIR/sing-quic" RE="$PROXY_DIR/REALITY"
PANEL_REPO=cedar2025/Xboard PANEL_BRANCH=master
# 面板里与节点通信相关的路径（这些变了，节点可能要跟着改）
PANEL_PATHS=(
  app/Http/Controllers/V1/Server app/Http/Controllers/V2/Server
  app/Http/Routes/V1/ServerRoute.php app/Http/Routes/V2/ServerRoute.php
  app/Http/Middleware/Server.php app/Http/Middleware/ServerV2.php
  app/Services/NodeSyncService.php app/Services/NodeRegistry.php app/Services/ServerService.php
  app/WebSocket app/Console/Commands/NodeWebSocketServer.php app/Jobs/NodeUserSyncJob.php
  app/Http/Resources/NodeResource.php app/Models/Server.php
)

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
merged() { git -C "$1" merge-base --is-ancestor "$2" "${3:-main}" 2>/dev/null; }
# go.mod 里某模块的版本 -> 可用于 git 的 rev（伪版本取末尾 hash，否则就是 tag）
gomod_rev() { # $1 go.mod 内容所在文件或 -（stdin） $2 模块
  local v
  v=$(awk -v m="$2" '($1==m && $2 ~ /^v/) {print $2; exit} ($1=="require" && $2==m) {print $3; exit}' "$1")
  if [[ "$v" =~ [0-9]{14}-([0-9a-f]{12})$ ]]; then echo "${BASH_REMATCH[1]}"; else echo "$v"; fi
}
# main 已合入的最高正式版 tag
merged_tag() { git -C "$1" -c versionsort.suffix=- tag -l 'v*' --merged main --sort=-v:refname | grep -vE -- '-(alpha|beta|rc)' | head -1; }
latest_tag() { # $1 repo, $2 stable|any
  if [ "$2" = stable ]; then
    git -C "$1" -c versionsort.suffix=- tag -l 'v*' --sort=-v:refname | grep -vE -- '-(alpha|beta|rc)' | head -1
  else
    git -C "$1" -c versionsort.suffix=- tag -l 'v*' --sort=-v:refname | head -1
  fi
}
log_range() { git -C "$1" log --no-merges --format='      %h %ad %s' --date=short "$2" | head -"${3:-40}"; }
# 打印 ref 相对上次审阅的新提交
since_seen() { # $1 key $2 repo $3 ref $4 label [$5 最多显示条数]
  local key=$1 repo=$2 ref=$3 label=$4 max=${5:-40} cur old n
  cur=$(sha "$repo" "$ref") || { echo "  [$label] ref $ref 不存在"; return; }
  NOW[$key]=$cur
  old=${SEEN[$key]:-}
  if [ -z "$old" ]; then
    echo "  [$label] @ ${cur:0:12}（从未标记审阅，显示最近 5 条）"
    log_range "$repo" "$ref" 5
  elif [ "$old" = "$cur" ]; then
    echo "  [$label] 自上次审阅无变化"
  else
    n=$(git -C "$repo" rev-list --no-merges --count "$old..$ref" 2>/dev/null || echo "?")
    echo "  [$label] 自上次审阅 (${old:0:12}) 新增 $n 个提交："
    log_range "$repo" "$old..$ref" "$max"
    [ "$n" != "?" ] && [ "$n" -gt "$max" ] && echo "      ……（还有 $((n - max)) 条，用 git log 查看）"
  fi
}
local_state() { # 工作区/推送状态
  local s ab
  s=$(git -C "$1" status --porcelain | wc -l)
  ab=$(git -C "$1" rev-list --left-right --count main...origin/main 2>/dev/null | awk '{print "未推送 "$1" / 落后 origin "$2}')
  echo "  本地: 分支 $(git -C "$1" branch --show-current)，未提交改动 $s 个文件，$ab"
}
# GitHub 页面上 “N commits behind <fork 父仓库>” 的本地等价计算
github_behind() { # $1 repo $2 父仓库分支 ref $3 显示名
  local n; n=$(git -C "$1" rev-list --count "origin/main..$2" 2>/dev/null || echo "?")
  if [ "$n" = 0 ]; then echo "  GitHub 相对 fork 父仓库 $3：不落后 ✓"
  else echo "  GitHub 会显示落后 fork 父仓库 $3 $n 个提交（内容已吸收的话用 merge -s ours 记录，见 SKILL.md）"; fi
}

# ---------------------------------------------------------------------------
echo
echo "================ Xboard-Node（原作者 cedar2025/Xboard-Node dev） ================"
local_state "$XN"
n=$(git -C "$XN" rev-list --count main..upstream/dev)
echo "  upstream/dev 中尚未合入 main 的提交: $n"
[ "$n" != 0 ] && log_range "$XN" main..upstream/dev
since_seen XN_UPSTREAM "$XN" upstream/dev "upstream/dev"
for b in $(git -C "$XN" for-each-ref refs/remotes/upstream --format='%(refname:short)' | grep -vE '^upstream(/HEAD|/dev)?$'); do
  key="XN_BR_$(echo "${b#upstream/}" | tr -c 'A-Za-z0-9\n' '_' | tr a-z A-Z)"
  since_seen "$key" "$XN" "$b" "$b（其他分支）" 10
done
github_behind "$XN" upstream/dev "cedar2025/Xboard-Node:dev"
echo "  原作者 go.mod 使用的内核（看他是否换了补丁版本）:"
git -C "$XN" show upstream/dev:go.mod | grep -E '^replace .*(sing-box|xray-core|sing-quic|reality)' | sed 's/^/      /'

echo
echo "================ Xboard 面板（$PANEL_REPO $PANEL_BRANCH，只看节点通信相关路径） ================"
export GH_TOKEN="${GH_TOKEN:-$(gh auth token --user Hei-XiaoHu 2>/dev/null)}"
if panel_head=$(gh api "repos/$PANEL_REPO/commits/$PANEL_BRANCH" --jq .sha 2>/dev/null); then
  NOW[PANEL_HEAD]=$panel_head
  NOW[PANEL_SINCE]=$(date -u +%FT%TZ)
  since=${SEEN[PANEL_SINCE]:-$(date -u -d '-60 days' +%FT%TZ)}
  [ -z "${SEEN[PANEL_SINCE]:-}" ] && echo "  （从未标记审阅，显示最近 60 天）"
  echo "  自 $since 以来触及节点 API 的提交："
  for p in "${PANEL_PATHS[@]}"; do
    gh api "repos/$PANEL_REPO/commits?sha=$PANEL_BRANCH&since=$since&path=$p&per_page=50" \
      --jq '.[] | "      \(.sha[0:8]) \(.commit.committer.date[0:10]) \(.commit.message | split("\n")[0])"' 2>/dev/null
  done | sort -u -k2,2r -k1,1 | head -40 | { out=$(cat); [ -n "$out" ] && echo "$out" || echo "      （无）"; }
  echo "  查看某个提交：gh api repos/$PANEL_REPO/commits/<sha> --jq '.files[].filename'"
else
  echo "  （gh 查询失败，跳过面板检查）"
fi

echo
echo "================ Xray-core（主上游 XTLS/Xray-core；参考 cedar2025/Xray-core） ================"
local_state "$XR"
cur=$(merged_tag "$XR"); st=$(latest_tag "$XR" stable); any=$(latest_tag "$XR" any)
echo "  main 已合入: $cur    上游最新正式版: $st    最新(含预发布): $any"
if [ -n "$st" ] && ! merged "$XR" "$st"; then
  echo "  >>> 有新正式版未合入：$cur..$st 共 $(git -C "$XR" rev-list --no-merges --count "$cur..$st") 个提交"
  log_range "$XR" "$cur..$st"
fi
n=$(git -C "$XR" rev-list --no-merges --count "$st..upstream/main")
echo "  上游 main 在 $st 之后还有 $n 个未发布提交（留意安全修复，必要时可 cherry-pick）"
since_seen XR_UPSTREAM_MAIN "$XR" upstream/main "XTLS upstream/main"
since_seen XR_CEDAR "$XR" cedar/main "参考 cedar/main" 20
github_behind "$XR" cedar/main "cedar2025/Xray-core:main"
xr_reality=$(gomod_rev "$XR/go.mod" github.com/xtls/reality)
echo "  Xray main 的 go.mod 用 REALITY: $xr_reality"
if [ -n "$st" ] && ! merged "$XR" "$st"; then
  echo "  上游 $st 的 go.mod 用 REALITY: $(git -C "$XR" show "$st:go.mod" | gomod_rev - github.com/xtls/reality)"
fi

echo
echo "================ sing-box（主上游 SagerNet/sing-box；参考 cedar2025/sing-box） ================"
local_state "$SB"
cur=$(merged_tag "$SB"); st=$(latest_tag "$SB" stable); any=$(latest_tag "$SB" any)
echo "  main 已合入: $cur    上游最新正式版: $st    最新(含预发布): $any"
if [ -n "$st" ] && ! merged "$SB" "$st"; then
  echo "  >>> 有新正式版未合入：$cur..$st 共 $(git -C "$SB" rev-list --no-merges --count "$cur..$st") 个提交"
  log_range "$SB" "$cur..$st"
fi
n=$(git -C "$SB" rev-list --no-merges --count "$st..upstream/stable")
echo "  上游 stable 分支在 $st 之后还有 $n 个未发布提交"
[ "$n" != 0 ] && log_range "$SB" "$st..upstream/stable" 20
since_seen SB_UPSTREAM_STABLE "$SB" upstream/stable "SagerNet upstream/stable"
since_seen SB_UPSTREAM_TESTING "$SB" upstream/testing "SagerNet upstream/testing（下个大版本开发线，仅供参考）" 20
since_seen SB_CEDAR_TESTING "$SB" cedar/testing "参考 cedar/testing" 20
echo "  cedar 最近更新的分支（原作者补丁一般在 testing，留意是否出现新分支）:"
git -C "$SB" for-each-ref --sort=-committerdate refs/remotes/cedar --count=5 \
  --format='      %(refname:short) %(committerdate:short) %(subject)' | grep -v '^      cedar ' | head -4
github_behind "$SB" cedar/testing "cedar2025/sing-box:testing"
sb_quic=$(gomod_rev "$SB/go.mod" github.com/sagernet/sing-quic)
echo "  sing-box main 的 go.mod require sing-quic: $sb_quic"

echo
echo "================ sing-quic（主上游 SagerNet/sing-quic；目标是 sing-box 要求的提交） ================"
local_state "$SQ"
if merged "$SQ" "$sb_quic"; then echo "  已包含 sing-box 需要的 $sb_quic ✓"
else echo "  >>> 尚未合入 sing-box 需要的 $sb_quic"; fi
if [ -n "${st:-}" ] && ! merged "$SB" "$st"; then
  echo "  上游 sing-box $st 需要 sing-quic: $(git -C "$SB" show "$st:go.mod" | gomod_rev - github.com/sagernet/sing-quic)"
fi
n=$(git -C "$SQ" rev-list --count main..upstream/main)
echo "  upstream/main 领先 main $n 个提交（一般跟随 sing-box 的版本，安全修复可提前合）"
since_seen SQ_UPSTREAM "$SQ" upstream/main "SagerNet upstream/main"
github_behind "$SQ" upstream/main "SagerNet/sing-quic:main"

echo
echo "================ REALITY（主上游 XTLS/REALITY；目标是 Xray 要求的提交） ================"
local_state "$RE"
if merged "$RE" "$xr_reality"; then echo "  已包含 Xray 需要的 $xr_reality ✓"
else echo "  >>> 尚未合入 Xray 需要的 $xr_reality"; fi
n=$(git -C "$RE" rev-list --count main..upstream/main)
echo "  upstream/main 领先 main $n 个提交"
since_seen RE_UPSTREAM "$RE" upstream/main "XTLS upstream/main"
github_behind "$RE" upstream/main "XTLS/REALITY:main"

echo
echo "================ Xboard-Node go.mod replace 是否指向各 fork 的 main ================"
for pair in "github.com/xtls/xray-core:$XR" "github.com/sagernet/sing-box:$SB" "github.com/sagernet/sing-quic:$SQ" "github.com/xtls/reality:$RE"; do
  mod=${pair%%:*} dir=${pair#*:}
  v=$(awk -v m="$mod" '$1=="replace" && $2==m {print $5; exit}' "$XN/go.mod")
  h=${v##*-}
  head=$(git -C "$dir" rev-parse --short=12 origin/main)
  case "$v" in v0.0.0-*) ;; *) echo "  !!! $mod 的伪版本不是 v0.0.0-… 形式：$v" ;; esac
  if [ "$h" = "$head" ]; then echo "  $mod -> $h ✓ (= origin/main)"
  elif [ "$(git -C "$dir" rev-parse "$h^{tree}" 2>/dev/null)" = "$(git -C "$dir" rev-parse origin/main^{tree})" ]; then
    echo "  $mod -> $h（内容与 origin/main $head 相同）✓"
  else echo "  $mod -> $h ≠ origin/main $head"; fi
done
sbq=$(awk '$1=="replace" && $2=="github.com/sagernet/sing-quic" {print $5}' "$SB/go.mod")
xnq=$(awk '$1=="replace" && $2=="github.com/sagernet/sing-quic" {print $5}' "$XN/go.mod")
[ "$sbq" = "$xnq" ] && echo "  sing-quic replace 在 sing-box 与 Xboard-Node 一致 ✓" \
  || echo "  !!! sing-quic replace 不一致：sing-box=$sbq Xboard-Node=$xnq"

if [ "$MARK" = 1 ]; then
  {
    echo "# upstream-sync 已审阅位置（status.sh --mark 自动生成，$(date -u +%FT%TZ)）"
    for k in $(printf '%s\n' "${!NOW[@]}" | sort); do echo "$k=${NOW[$k]}"; done
  } > "$STATE"
  echo
  echo "已写入审阅标记: $STATE"
fi
