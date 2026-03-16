package ctxbuild

import "strings"

// ResolveChain expands a slash-separated department path into an ordered
// slice of all ancestor paths from top-level to the target department.
//
// Examples:
//
//	ResolveChain("")                   → []
//	ResolveChain("engineering")        → ["engineering"]
//	ResolveChain("engineering/backend")→ ["engineering", "engineering/backend"]
func ResolveChain(deptPath string) []string {
	deptPath = strings.TrimSpace(deptPath)
	if deptPath == "" {
		return nil
	}
	parts := strings.Split(deptPath, "/")
	chain := make([]string, len(parts))
	for i, part := range parts {
		if i == 0 {
			chain[i] = strings.TrimSpace(part)
		} else {
			chain[i] = chain[i-1] + "/" + strings.TrimSpace(part)
		}
	}
	return chain
}
