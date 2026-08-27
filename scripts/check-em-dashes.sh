#!/bin/bash

set -e

BASE_REF="${1:-origin/master}"

# AGENTS.md bans the em dash: rewrite the clause rather than swapping the dash
# for a hyphen, so a paired aside becomes commas or parentheses and a dash
# introducing an explanation becomes a colon or a new sentence.
#
# This compares counts per file rather than reading the diff, and the difference
# matters in three ways. A reflow re-adds every line it touches without writing
# any prose, and a diff would report the whole backlog as new. An em dash added
# in one commit and rewritten in a later one is not in the change at all, and a
# diff would still see it. And what a reviewer cares about is the state the
# branch ends in, not the route it took.
#
# The backlog is left alone deliberately. Clearing it means rewriting each
# clause rather than running a substitution, so nothing new joins it and a file
# already being edited for another reason is the right moment to clear what it
# carries.
#
# Lockfiles and generated output are excluded: nothing writes prose there and
# the tools that own them would put it back.
DASH=$'—'

# A bare "—" is a data placeholder, a missing value in a table cell rather than
# prose, and the rule leaves those alone. Stripping them before counting is what
# tells the two apart.
count_in() {
  local ref="$1" file="$2"
  git show "${ref}:${file}" 2>/dev/null | sed "s/\"${DASH}\"//g" | grep -o -- "$DASH" | wc -l
}

CHANGED=$(git diff --name-only "$BASE_REF...HEAD" -- \
  ':(exclude)*lock.yaml' \
  ':(exclude)*lock.json' \
  ':(exclude)*.gen.ts' \
  ':(exclude)public/app.css' \
  ':(exclude)*.snap' || true)

WORSE=""
for FILE in $CHANGED; do
  [ -f "$FILE" ] || continue
  BEFORE=$(count_in "$BASE_REF" "$FILE")
  AFTER=$(count_in HEAD "$FILE")
  if [ "$AFTER" -gt "$BEFORE" ]; then
    WORSE="${WORSE}  ${FILE}: ${BEFORE} to ${AFTER}"$'\n'
  fi
done

if [ -z "$WORSE" ]; then
  exit 0
fi

echo
echo "This change adds em dashes:"
echo
printf '%s' "$WORSE"
echo
echo "Rewrite the clause rather than swapping the dash for a hyphen. A paired"
echo "aside becomes commas or parentheses; a dash introducing an explanation"
echo "becomes a colon or a new sentence."
echo
echo "A bare \"${DASH}\" standing alone is a data placeholder and is allowed."
echo
exit 1
