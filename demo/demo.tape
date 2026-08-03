# VHS tape for rss-readers — https://github.com/charmbracelet/vhs
# Record with:  vhs demo/demo.tape   (run from the repo root)

Output docs/demo.gif

Set Shell "zsh"
Set FontSize 15
Set Width 1280
Set Height 800
Set Padding 24
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 55ms

# Launch the reader (cache is warm, so the fetch page resolves quickly).
Hide
Type "clear"
Enter
Show
Sleep 600ms
Type "./rss-readers"
Sleep 400ms
Enter
Sleep 2500ms

# Browse the aggregated list.
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 700ms

# Filter by feed with tab — watch the sidebar highlight move.
Tab
Sleep 1100ms
Tab
Sleep 1300ms

# Read a full article and scroll through it.
Enter
Sleep 1800ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 1100ms
Escape
Sleep 700ms

# Back to all feeds.
Shift+Tab
Sleep 500ms
Shift+Tab
Sleep 800ms

# Fuzzy search.
Type "/"
Sleep 400ms
Type "rust"
Sleep 1500ms
Escape
Sleep 700ms

# Help overlay.
Type "?"
Sleep 2200ms
Escape
Sleep 600ms

# Quit.
Type "q"
Sleep 800ms
