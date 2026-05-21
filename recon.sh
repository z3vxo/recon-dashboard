#!/bin/bash
set -euo pipefail



usage() {
    echo "Usage: $0 [--large-target] [--only-subs] [--no-mutate] [--no-sec-pass] [--no-path-probe] <domain>"
}

ONLY_SUBS=false
ACTIVE_FLAGS=()
HTTPX_FLAGS=()
DOMAIN=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --large-target)
            ACTIVE_FLAGS+=("--no-sec-pass" "--no-mutate")
            HTTPX_FLAGS+=("--no-path-probe")
            shift
            ;;
        --only-subs)
            ONLY_SUBS=true
            shift
            ;;
        --no-mutate|--no-sec-pass)
            ACTIVE_FLAGS+=("$1")
            shift
            ;;
        --no-path-probe)
            HTTPX_FLAGS+=("$1")
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --*)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
        *)
            if [[ -n "$DOMAIN" ]]; then
                echo "Unexpected argument: $1" >&2
                usage >&2
                exit 1
            fi
            DOMAIN=$1
            shift
            ;;
    esac
done

if [[ -z "$DOMAIN" ]]; then
    usage >&2
    exit 1
fi

die() { echo -e "\033[1;31m[!]\033[0m $*" >&2; exit 1; }

bash recon-files/subdomain2.sh "$DOMAIN"       || die "Stage 1 (passive enumeration) failed"
[[ -s "$HOME/.recon/$DOMAIN/subdomains/all_subs.txt" ]]   || die "Stage 1 produced no subdomains — aborting"

bash recon-files/subdomains_active.sh "${ACTIVE_FLAGS[@]}" "$DOMAIN" || die "Stage 2 (active DNS) failed"
[[ -s "$HOME/.recon/$DOMAIN/subdomains/final_subs.txt" ]] || die "Stage 2 produced no resolved subdomains — aborting"

if [[ "$ONLY_SUBS" == true ]]; then
    echo -e "\033[1;32m[*]\033[0m --only-subs set; skipping Stage 3 (HTTP probe)"
    exit 0
fi

bash recon-files/alive_httpx_probe.sh "${HTTPX_FLAGS[@]}" "$DOMAIN" || die "Stage 3 (HTTP probe) failed"
#python3 server/app.py
