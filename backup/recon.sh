#!/bin/bash
set -euo pipefail



# for codex
# i need to add flags, this pipeline falls apart on a target like westpac.com.au with thousands of hosts
# need to add flags
# --only-subs -> only collects subdomains can be mixed with
#    --no-mutate -> skips host mutation step
#    --no-sec-pass -> skips 2nd level subfinder pass
#    note, keep the same flow tho if both flags are present,
#    just run subdomain enum(subdomains2.sh) -> subomains_active.sh resolve that list
if [ -z "$1" ]; then
    echo "Usage: $0 <domain>"
    exit 1
fi

DOMAIN=$1

die() { echo -e "\033[1;31m[!]\033[0m $*" >&2; exit 1; }

bash recon-files/subdomain2.sh "$DOMAIN"       || die "Stage 1 (passive enumeration) failed"
[[ -s "$HOME/.recon/$DOMAIN/subdomains/all_subs.txt" ]]   || die "Stage 1 produced no subdomains — aborting"

bash recon-files/subdomains_active.sh "$DOMAIN" || die "Stage 2 (active DNS) failed"
[[ -s "$HOME/.recon/$DOMAIN/subdomains/final_subs.txt" ]] || die "Stage 2 produced no resolved subdomains — aborting"

bash recon-files/alive_httpx_probe.sh "$DOMAIN" || die "Stage 3 (HTTP probe) failed"
#python3 server/app.py
