#!/bin/bash

set -o errexit
set -o pipefail

main() {
  _cd_into_top_level
  _generate_coverage_files
  _combine_coverage_reports
}

_cd_into_top_level() {
  cd "$(git rev-parse --show-toplevel)"
}

_generate_coverage_files() {
  for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -type d); do
    if ls $dir/*.go &>/dev/null ; then
      go test -covermode=count -coverprofile=$dir/profile.coverprofile $dir || fail=1
    fi
  done
}


_combine_coverage_reports() {
  gover
}

main "$@"