#!/usr/bin/env bash
# Produce a 4K (3840x2160, 16:9), Twitter/X-ready MP4 of the reader.
#
# Why not record 4K directly with VHS? VHS renders frames in real time, and a
# 3840x2160 grid is too heavy to sustain the target framerate — it drops frames
# to keep wall-clock, which collapses the pauses. So we record at a resolution
# VHS renders cleanly (exact timing) and upscale with ffmpeg (lanczos), which
# preserves timing and keeps the large terminal glyphs crisp.
#
# Requirements: vhs, ffmpeg. Run from anywhere.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> Recording with VHS (docs/demo.gif)"
vhs demo/demo.tape

echo "==> Upscaling to 4K MP4 (demo/rss-readers-4k.mp4)"
ffmpeg -y -i docs/demo.gif \
  -vf "fps=30,scale=3840:2160:force_original_aspect_ratio=decrease:flags=lanczos,pad=3840:2160:(ow-iw)/2:(oh-ih)/2:color=0x1a1b26,format=yuv420p" \
  -c:v libx264 -crf 18 -preset slow -movflags +faststart -an \
  demo/rss-readers-4k.mp4

echo "==> Done"
ffprobe -v error -select_streams v:0 \
  -show_entries stream=width,height,codec_name,pix_fmt \
  -show_entries format=duration -of default=noprint_wrappers=1 \
  demo/rss-readers-4k.mp4
