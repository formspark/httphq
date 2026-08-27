#!/bin/bash

set -e -o pipefail

# Reports the worst score a ratchet rule finds, which is the number its cap
# should be set to. The caps in the eslint config are measured this way rather
# than chosen, so this is how to re-derive one after a refactor.
#
#   scripts/measure-ratchet.sh complexity        # every linted file
#   scripts/measure-ratchet.sh complexity ts     # modules only
#   scripts/measure-ratchet.sh complexity tsx    # components only
#   scripts/measure-ratchet.sh max-params
#
# It exists because doing this by hand is quietly unreliable, in two ways that
# both look like a clean tree.
#
# Each rule words its message differently. complexity says "has a complexity of
# 15"; max-params and max-depth say "(3)". A grep written for one shape matches
# nothing for the other and prints no findings.
#
# And a glob handed to eslint is not a glob the shell expanded. Passing
# 'apps/**/*.tsx' can match nothing at all while looking like it scoped
# correctly. So scope is an extension here, and the file list comes from git.

RULE="${1:?usage: measure-ratchet.sh <rule> [extension]}"
EXTENSION="${2:-}"

case "$RULE" in
  # complexity takes an options object; the others take a bare number. The
  # variant has to match the config or the number is not the one the cap is
  # compared against.
  complexity) CONFIG='{"'"$RULE"'":["error",{"max":0,"variant":"modified"}]}' ;;
  *) CONFIG='{"'"$RULE"'":["error",0]}' ;;
esac

run_eslint() {
  if [ -z "$EXTENSION" ]; then
    pnpm exec eslint . --no-warn-ignored --rule "$CONFIG" 2>/dev/null || true
  else
    git ls-files "*.${EXTENSION}" \
      | xargs pnpm exec eslint --no-warn-ignored --rule "$CONFIG" 2>/dev/null || true
  fi
}

if [ -n "$EXTENSION" ] && [ -z "$(git ls-files "*.${EXTENSION}")" ]; then
  echo "No tracked .${EXTENSION} files." >&2
  exit 1
fi

# "has a complexity of 15" and "too many parameters (3)" are the two shapes.
WORST=$(run_eslint \
  | grep -oE "(of [0-9]+|\([0-9]+\))\.?( Maximum| +$RULE)" \
  | grep -oE '[0-9]+' \
  | sort -n | tail -1)

if [ -z "$WORST" ]; then
  echo "No findings for '$RULE'." >&2
  echo >&2
  echo "That is almost never a clean tree, because the rule was run at a cap of" >&2
  echo "zero. Check the rule name, and that this script knows how the rule" >&2
  echo "words its message. A silent empty result is the failure this refuses" >&2
  echo "to print." >&2
  exit 1
fi

echo "$WORST"
