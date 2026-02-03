#!/bin/bash

# Setup NFTables rules for packet forwarding to NFQueue

set -e

if [ "$EUID" -ne 0 ]; then 
    echo "This script must be run as root"
    exit 1
fi

echo "Configuring NFTables for packet interception..."

# Delete existing table if it exists (ignore errors)
echo "Cleaning up existing rules..."
nft delete table ip netvine-table 2>/dev/null || echo "No existing table to delete"

# Create fresh table
echo "Creating new table and chain..."
nft add table ip netvine-table

# Create chain
nft add chain ip netvine-table base-rule-chain { type filter hook forward priority 0\; policy accept\; }

# Add queue rule
nft add rule ip netvine-table base-rule-chain queue num 0-3 bypass

echo "NFTables configured successfully!"
echo ""
echo "To view rules: sudo nft list ruleset"
echo "To delete rules: sudo nft delete table ip netvine-table"
