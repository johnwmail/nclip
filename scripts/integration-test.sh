#!/bin/bash
set -euo pipefail

# Minimal delegator: run the test suite under scripts/integration/
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/integration"
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    bash "$DIR/run_all.sh"
fi