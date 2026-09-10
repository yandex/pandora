#!/bin/bash

set -o errexit
set -o pipefail

main() {
  local fail=0
  _cd_into_top_level
  _generate_coverage_files || fail=1
  if [ "$fail" -eq 1 ]; then
    echo "There was an error while generating coverage files."
    exit 1
  fi
  _combine_coverage_reports
}

_cd_into_top_level() {
  local top_level_dir
  top_level_dir="$(git rev-parse --show-toplevel)"
  cd "$top_level_dir" || {
    echo "Unable to change to project root directory." >&2
    exit 1
  }
}

_generate_coverage_files() {
  local dir fail=0
  while IFS= read -r -d '' dir; do
    if ls "$dir"/*.go &>/dev/null; then
      go test -covermode=count -coverprofile="$dir/profile.coverprofile" "$dir" || fail=1
    fi
  done < <(find . -maxdepth 10 -not -path './.git*' -not -path '*/vendor/*' -not -path '*/mocks/*' -type d -print0)
  return "$fail"
}

_combine_coverage_reports() {
  gover
}

main "$@"
