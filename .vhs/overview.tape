Source ".vhs/_settings.tape"
Output "docs/assets/overview.gif"

Require chezit
Require chezmoi

Type "chezit"
Sleep 1s
Enter
Sleep 3s

# Landing page — let viewer see the summary box, then enter Status
Screenshot "docs/assets/landing.png"
Enter
Sleep 2s

# Status tab — panel open with diff view (default for Status)
# Move onto the first Local Drift rows
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Sleep 1s

# Screenshot captures the sync header, Local Drift, and Unstaged sections
Screenshot "docs/assets/status.png"
Sleep 200ms
Type "v"
Sleep 1.5s

# Walk down the drift rows with the content view
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Sleep 1s

# Continue to the end of Local Drift and into Unstaged
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Sleep 1s

# Switch to Files tab — panel resets to content view (default for Files)
Type "2"
Sleep 2s

# Navigate the file tree — expanded structure with ghostty, homebrew, ripgrep
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "l"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms

# Screenshot on .config/chezit/config.yaml (4th j)
Screenshot "docs/assets/files.png"

Type "j"
Sleep 400ms
Type "l"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Sleep 1.5s

# Switch to Info tab
Type "3"
Sleep 2s

# Cycle through Info sub-views: Config → Full → Data → Doctor
Right
Sleep 1.5s
Right
Sleep 1.5s
Right
Sleep 2s
Screenshot "docs/assets/info.png"

# Switch to Commands tab
Tab
Sleep 2s
Screenshot "docs/assets/commands.png"
Sleep 1s

# Quit
Type "q"
Sleep 1s
