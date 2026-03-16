#!/bin/bash

# Dogfood example: Propose web search tool

echo "Running AegisCourt dogfood example..."

# Assume init already done
# ./aegiscourt setup init --non-interactive

# Start runtime
./aegiscourt runtime start

# Propose tool
./aegiscourt governance propose add-tool web_search

# Wait for court (in real, would wait)
sleep 5

# View proposal
./aegiscourt governance court view 1

# Vote approve (auto mode)
echo "Auto-approving in hobbyist mode..."

echo "Dogfood complete. Check audit log for full trace."