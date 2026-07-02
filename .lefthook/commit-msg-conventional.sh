#!/usr/bin/env bash
# Conventional Commits subject check, called by lefthook's commit-msg hook.
# Same contract as the arfpg-spec-workflow reference .githooks/commit-msg.
set -euo pipefail

# First non-comment, non-empty line is the subject.
subject=""
while IFS= read -r line; do
  case "$line" in
    \#*) continue ;;
    "") [ -z "$subject" ] && continue || break ;;
    *) subject="$line"; break ;;
  esac
done < "$1"

if [ -z "$subject" ]; then
  echo "commit-msg: empty commit subject" >&2
  exit 1
fi

pattern='^(feat|fix|docs|style|refactor|perf|test|build|chore|ci|revert)(\(.+\))?: .{1,72}$'

if ! [[ "$subject" =~ $pattern ]]; then
  echo "commit-msg: subject must match Conventional Commits:" >&2
  echo "  <type>(<scope>): <subject>" >&2
  echo "types: feat fix docs style refactor perf test build chore ci revert" >&2
  echo "subject: imperative, lowercase, no trailing period, ≤72 chars" >&2
  exit 1
fi

exit 0
