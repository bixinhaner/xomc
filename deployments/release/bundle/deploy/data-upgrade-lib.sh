#!/usr/bin/env bash

# should_refresh_builtin <new-file> <live-file> <previous-baseline-file>
# 返回 0 表示可用新版刷新，返回 1 表示现网文件相对上一版基线已有运维修改，必须保留。
# 老版本没有可比较基线时沿用既有刷新行为，避免历史 builtin 永久停留在旧版本。
should_refresh_builtin() {
  local new_file="$1" live_file="$2" previous_file="$3"
  [ -f "$new_file" ] && [ -f "$live_file" ] || return 1
  if [ -f "$previous_file" ] && ! cmp -s "$live_file" "$previous_file"; then
    return 1
  fi
  ! cmp -s "$new_file" "$live_file"
}
