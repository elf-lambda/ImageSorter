# Go Image Sorter

A minimal Go-powered image browser and tagger for local photo and video folders. It scans a directory of images and `.webm` or `.gif` files, generates thumbnails, stores metadata in SQLite, and serves a simple frontend for viewing and tagging.

## Features

-   📁 Auto-index images and `.webm` | `.gif` files from a folder
-   🏷️ Add/remove tags (stored in SQLite)
-   🔍 Search images by tag
-   🖼️ Lazy-loading image previews (via thumbnails)
-   🎞️ `.webm` playback with overlay controls
-   📋 Copy filename or full path for selected media
-   🧹 Rename all files to unix timestamp name hash
-   📁 Generate thumbnails using FFMPeg

## Usage

1. Setup your image directory in main.go
2. Build and run the Go server.
3. Access the app in your browser at `http://localhost:8080`.
4. Use the UI to browse, preview, tag, and search your media.

## Setup

```bash
go run ./src/
```
