package chezmoi

import "strings"

// ParseStatus parses `chezmoi status` output.
func ParseStatus(output string) []FileStatus {
	var files []FileStatus
	for line := range strings.SplitSeq(output, "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, FileStatus{
			SourceStatus: rune(line[0]),
			DestStatus:   rune(line[1]),
			Path:         line[3:],
		})
	}
	return files
}

// ParseGitPorcelain parses `git status --porcelain` output.
func ParseGitPorcelain(output string) (staged, unstaged []GitFile, err error) {
	for line := range strings.SplitSeq(output, "\n") {
		if len(line) < 4 {
			continue
		}
		x := line[0]
		y := line[1]
		path := line[3:]

		if len(path) >= 2 && path[0] == '"' && path[len(path)-1] == '"' {
			path = path[1 : len(path)-1]
		}

		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}

		if x != ' ' && x != '?' {
			staged = append(staged, GitFile{
				Path:       path,
				StatusCode: string(x),
			})
		}

		if y != ' ' {
			code := string(y)
			if x == '?' && y == '?' {
				code = "U"
			}
			unstaged = append(unstaged, GitFile{
				Path:       path,
				StatusCode: code,
			})
		}
	}
	return staged, unstaged, nil
}

// ParseGitLogOneline parses `git log --oneline` output.
func ParseGitLogOneline(output string) []GitCommit {
	var commits []GitCommit
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		hash, message, _ := strings.Cut(line, " ")
		commits = append(commits, GitCommit{Hash: hash, Message: message})
	}
	return commits
}
