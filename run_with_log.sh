#!/bin/bash
# Run emulator and log all output to a timestamped file
# Usage: ./run_with_log.sh -rom demo.rom -scale 3

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="emulator_log_${TIMESTAMP}.txt"

echo "=== Emulator Log Started: $(date '+%Y-%m-%d %H:%M:%S') ===" | tee "$LOG_FILE"
echo "Logging to: $LOG_FILE" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

./nitro-core-dx "$@" 2>&1 | tee -a "$LOG_FILE"

echo "" | tee -a "$LOG_FILE"
echo "=== Emulator Log Ended: $(date '+%Y-%m-%d %H:%M:%S') ===" | tee -a "$LOG_FILE"
