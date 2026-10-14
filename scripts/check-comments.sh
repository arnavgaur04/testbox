#!/bin/sh
# Check Go comments: max 2 consecutive lines, max 40 chars of text.
# Text = line content after leading // markers. Skips //go: directives.
# Usage: sh scripts/check-comments.sh [dir...]
set -u

MAX_TEXT=40
MAX_LINES=2
fail=0

check_file() {
  f="$1"
  consec=0
  lineno=0
  while IFS= read -r line || [ -n "$line" ]; do
    lineno=$((lineno + 1))
    stripped=$(printf '%s' "$line" | sed -e 's/^[[:space:]]*//')
    case "$stripped" in
      "//go:"*) consec=0; continue ;;
      "//"*)
        text=$(printf '%s' "$stripped" | sed -e 's|^///*[[:space:]]*||')
        consec=$((consec + 1))
        n=$(printf '%s' "$text" | wc -m)
        # wc -m counts trailing newline as 1 when piped via printf %s? No newline added. Use as-is.
        if [ "$n" -gt "$MAX_TEXT" ]; then
          echo "FAIL $f:$lineno: comment text $n chars (max $MAX_TEXT): $line"
          fail=1
        fi
        if [ "$consec" -gt "$MAX_LINES" ]; then
          echo "FAIL $f:$lineno: comment block exceeds $MAX_LINES lines"
          fail=1
        fi
        ;;
      *) consec=0 ;;
    esac
  done < "$f"
}

if [ "$#" -eq 0 ]; then
  set -- cmd internal pkg
fi

for d in "$@"; do
  [ -d "$d" ] || continue
  find "$d" -name '*.go' > /tmp/comment_check_list.txt
  while IFS= read -r f; do
    check_file "$f"
  done < /tmp/comment_check_list.txt
done
rm -f /tmp/comment_check_list.txt

exit "$fail"
