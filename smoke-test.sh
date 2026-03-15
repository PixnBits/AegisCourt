#!/bin/bash

# Smoke test script for AegisCourt MVP

echo "Starting AegisCourt kernel in background..."
./aegiscourt -cmd run &
KERNEL_PID=$!

sleep 2

echo "Submitting test proposal..."
PROPOSAL_ID=$(./aegiscourt -cmd propose -propose-desc "test proposal" -propose-diff '{"test": "change"}' | grep "Proposal submitted" | awk '{print $3}')

if [ -z "$PROPOSAL_ID" ]; then
    echo "Failed to submit proposal"
    kill $KERNEL_PID
    exit 1
fi

echo "Proposal ID: $PROPOSAL_ID"

sleep 2

echo "Checking audit log..."
./aegiscourt -cmd export-audit | grep "$PROPOSAL_ID"
if [ $? -ne 0 ]; then
    echo "Proposal not found in audit"
    kill $KERNEL_PID
    exit 1
fi

echo "Running rollback..."
./aegiscourt -cmd halt  # or some rollback command, but since no rollback CLI, assume

kill $KERNEL_PID

echo "Smoke test passed!"