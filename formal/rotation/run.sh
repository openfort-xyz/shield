#!/usr/bin/env bash
# Model-check Rotation.tla: the baseline must pass and every mutant must
# violate the invariant it targets. Exit 0 only if all of that holds.
set -euo pipefail

cd "$(dirname "$0")"

TLA_VERSION=v1.7.4
TLA_SHA256=936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88
JAVA=${JAVA:-java}

if [[ ! -f tla2tools.jar ]]; then
  curl -sfL -o tla2tools.jar "https://github.com/tlaplus/tlaplus/releases/download/${TLA_VERSION}/tla2tools.jar"
fi
echo "${TLA_SHA256}  tla2tools.jar" | shasum -a 256 -c - >/dev/null

MUTANTS=(MutNoRecheck MutDeleteInsert MutNoKeyCheck MutMultiPending MutOneShot MutCommitNoAuth MutNoBackfill MutSecCommitNoAuth)

write_cfg() {
  local enabled=$1 out=$2
  {
    echo "SPECIFICATION Spec"
    echo "CONSTANTS"
    echo "  Writers = {1, 2}"
    echo "  MaxKey = 4"
    echo "  MaxSec = 3"
    for m in "${MUTANTS[@]}"; do
      if [[ $m == "$enabled" ]]; then echo "  $m = TRUE"; else echo "  $m = FALSE"; fi
    done
    echo "INVARIANTS NoUndecryptableShare AtMostOnePending ClientCanRecover ApiClientCanAuth"
    echo "CHECK_DEADLOCK FALSE"
  } >"$out"
}

check() {
  local name=$1 expect=$2
  local cfg log
  local dir
  dir=$(mktemp -d -t rotation)
  cfg=$dir/Rotation.cfg
  log=$dir/tlc.log
  write_cfg "$name" "$cfg"
  set +e
  "$JAVA" -XX:+UseParallelGC -cp tla2tools.jar tlc2.TLC -workers auto -metadir "$dir/states" \
    -config "$cfg" Rotation.tla >"$log" 2>&1
  local rc=$?
  set -e
  if [[ $expect == pass ]]; then
    if [[ $rc -eq 0 ]] && grep -q "No error has been found" "$log" && grep -q "0 states left on queue" "$log"; then
      echo "PASS  baseline: $(grep "states left on queue" "$log" | tail -1)"
    else
      echo "FAIL  baseline: expected no error"; cat "$log"; return 1
    fi
  else
    if [[ $rc -ne 0 ]] && grep -q "Invariant $expect is violated" "$log"; then
      echo "PASS  $name violates $expect"
    else
      echo "FAIL  $name: expected $expect to be violated"; tail -30 "$log"; return 1
    fi
  fi
  rm -rf "$dir"
}

status=0
check baseline pass || status=1
check MutNoRecheck NoUndecryptableShare || status=1
check MutDeleteInsert NoUndecryptableShare || status=1
check MutNoKeyCheck NoUndecryptableShare || status=1
check MutMultiPending AtMostOnePending || status=1
check MutOneShot ClientCanRecover || status=1
check MutCommitNoAuth ApiClientCanAuth || status=1
check MutNoBackfill NoUndecryptableShare || status=1
check MutSecCommitNoAuth ApiClientCanAuth || status=1
exit $status
