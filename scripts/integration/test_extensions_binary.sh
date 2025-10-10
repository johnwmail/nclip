#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_binary_archive_extensions() {
    log "Testing binary/archive file extensions in Content-Disposition..."
    local failed_tests=0
    # application/zip -> .zip
    log "Testing application/zip -> .zip"
    local zip_file="/tmp/nclip_test_ext.zip"
    echo -n -e "PK\x03\x04testzip" > "$zip_file"
    local response
    try_post response "$NCLIP_URL/" "@${zip_file}" -H "Content-Type: application/zip"
    rm -f "$zip_file"
    if [[ -z "$response" || ! "$response" =~ http ]]; then
        error "Failed to upload application/zip content. Response: $response"
        ((failed_tests++))
    else
        slug=$(basename "$response")
        raw_url="$NCLIP_URL/raw/$slug"
        header=$(cget -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition")
        if [[ "$header" == *"$slug.zip"* ]]; then
            success "application/zip extension correctly appended: .zip"
        else
            error "application/zip extension NOT appended correctly. Header: $header"
            ((failed_tests++))
        fi
    fi

    # image/png -> .png
    log "Testing image/png -> .png"
    local png_file="/tmp/nclip_test_ext.png"
    echo -n -e "\x89PNGtestpng" > "$png_file"
    try_post response "$NCLIP_URL/" "@${png_file}" -H "Content-Type: image/png"
    rm -f "$png_file"
    if [[ -z "$response" || ! "$response" =~ http ]]; then
        error "Failed to upload image/png content. Response: $response"
        ((failed_tests++))
    else
        slug=$(basename "$response")
        raw_url="$NCLIP_URL/raw/$slug"
        header=$(cget -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition")
        if [[ "$header" == *"$slug.png"* ]]; then
            success "image/png extension correctly appended: .png"
        else
            error "image/png extension NOT appended correctly. Header: $header"
            ((failed_tests++))
        fi
    fi

    # application/pdf -> .pdf
    log "Testing application/pdf -> .pdf"
    local pdf_file="/tmp/nclip_test_ext.pdf"
    echo -n -e "%PDF-1.4testpdf" > "$pdf_file"
    if try_post response "$NCLIP_URL/" "@${pdf_file}" -H "Content-Type: application/pdf"; then
        :
    fi
    rm -f "$pdf_file"
    if [[ -z "$response" || ! "$response" =~ http ]]; then
        error "Failed to upload application/pdf content. Response: $response"
        ((failed_tests++))
    else
        slug=$(basename "$response")
        raw_url="$NCLIP_URL/raw/$slug"
        header=$(cget -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition")
        if [[ "$header" == *"$slug.pdf"* ]]; then
            success "application/pdf extension correctly appended: .pdf"
        else
            error "application/pdf extension NOT appended correctly. Header: $header"
            ((failed_tests++))
        fi
    fi

    if [[ $failed_tests -eq 0 ]]; then success "All binary/archive extension tests passed"; return 0; else error "$failed_tests binary/archive extension test(s) failed"; return 1; fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_binary_archive_extensions
    cleanup_temp_files
fi
