package avatar

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/url"
	"strings"
	"time"
)

var backgrounds = []string{
	"b6e3f4",
	"c0aede",
	"d1d4f9",
	"ffd5dc",
	"ffdfbf",
}

// RandomURL returns a random-looking cartoon human avatar URL for an agent.
func RandomURL(project, agent string) string {
	seed := strings.Trim(strings.Join([]string{project, agent, randomSuffix()}, "-"), "-")
	return URL(seed)
}

func URL(seed string) string {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		seed = randomSuffix()
	}
	bg := backgrounds[int(hash(seed))%len(backgrounds)]
	return fmt.Sprintf(
		"https://api.dicebear.com/9.x/notionists/svg?seed=%s&backgroundColor=%s",
		url.QueryEscape(seed),
		bg,
	)
}

func randomSuffix() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
