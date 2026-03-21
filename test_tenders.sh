#!/bin/bash

# Add sample tenders via API (you'll need to create a POST endpoint for tenders)
# For now, let's check the database directly

# Check if the database file exists
if [ -f "tinymuscle.db" ]; then
    echo "Database file exists. You can inspect it with: sqlite3 tinymuscle.db 'SELECT * FROM tenders;'"
else
    echo "No database file yet. The backend will create it when data is added."
fi