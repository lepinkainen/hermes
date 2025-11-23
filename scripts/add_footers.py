#!/usr/bin/env python3
"""
Script to add or update document footers with creation and review dates.

This script adds a standardized footer to all markdown files in the docs/ directory:
- Uses file creation time for "Document created" date
- Uses file modification time for "Last reviewed" date
- Updates existing footers instead of duplicating them

Usage:
    python3 scripts/add_footers.py
"""

import os
import re
from datetime import datetime
from pathlib import Path

def get_file_dates(filepath):
    """Get file creation and modification dates in YYYY-MM-DD format."""
    stat = filepath.stat()
    # On macOS, st_birthtime is creation time, on Linux use st_ctime
    if hasattr(stat, 'st_birthtime'):
        created_time = datetime.fromtimestamp(stat.st_birthtime)
    else:
        created_time = datetime.fromtimestamp(stat.st_ctime)
    
    modified_time = datetime.fromtimestamp(stat.st_mtime)
    
    return created_time.strftime('%Y-%m-%d'), modified_time.strftime('%Y-%m-%d')

def has_footer(content):
    """Check if file already has the footer."""
    return '*Document created:' in content and '*Last reviewed:' in content

def add_or_update_footer(filepath):
    """Add or update footer in a markdown file."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    created_date, reviewed_date = get_file_dates(filepath)
    
    footer = f"""

---

*Document created: {created_date}*
*Last reviewed: {reviewed_date}*"""
    
    if has_footer(content):
        # Update existing footer
        content = re.sub(
            r'\n\n---\n\n\*Document created:.*?\*\n\*Last reviewed:.*?\*',
            footer,
            content,
            flags=re.DOTALL
        )
        print(f"Updated footer in {filepath}")
    else:
        # Add new footer
        content += footer
        print(f"Added footer to {filepath}")
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)

def main():
    docs_dir = Path('docs')
    if not docs_dir.exists():
        print("docs directory not found!")
        return
    
    # Find all markdown files
    md_files = list(docs_dir.rglob('*.md'))
    
    print(f"Found {len(md_files)} markdown files to process...")
    
    for filepath in md_files:
        try:
            add_or_update_footer(filepath)
        except Exception as e:
            print(f"Error processing {filepath}: {e}")
    
    print("Done!")

if __name__ == '__main__':
    main()