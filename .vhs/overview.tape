Source ".vhs/_settings.tape"
Output "docs/assets/overview.gif"

Require chezit
Require chezmoi

Type "chezit"
Sleep 1s
Enter
Sleep 2s

# Landing page — let viewer see the summary box, then enter Status
Screenshot "docs/assets/landing.png"
Enter
Sleep 2s

# Status tab — panel open with diff view (default for Status)
# Navigate through drift items (2 items: zshrc, ghostty)
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Sleep 1s

# Screenshot captures drift + unstaged + staged sections visible
Screenshot "docs/assets/status.png"
Sleep 200ms
Type "v"
Sleep 1.5s

# Navigate through unstaged section (3 items: ripgreprc, starship, ssh)
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Sleep 1s

# Navigate through staged section (3 items: bashrc, brew.env, zprofile)
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
