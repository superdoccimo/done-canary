package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func requirePolicyFragments(t *testing.T, relative string, fragments ...string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectRoot(t), relative))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Errorf("%s is missing creator-first policy fragment %q", relative, fragment)
		}
	}
}

func TestCreatorFirstPublicCoveragePolicy(t *testing.T) {
	requirePolicyFragments(t, "README.md",
		"## Don't erase the person who built it",
		"you get the traffic → the creator disappears",
		"earning money from useful coverage is not prohibited.",
		"person who introduces an open-source project is not the source",
		"https://github.com/superdoccimo/done-canary",
		"https://x.com/superdoccimo",
	)
	requirePolicyFragments(t, "ATTRIBUTION.md",
		"A blog post is not the canonical source.",
		"A viral X post is not the canonical source.",
		"A YouTube video is not the canonical source.",
		"A newsletter is not the canonical source.",
		"A mirror is not the canonical source.",
		"does not reduce the need for creator attribution.",
		"DISTRIBUTION_SUCCESS",
		"CREATOR_ATTRIBUTION_FAILURE",
		"CREATOR_ATTRIBUTION_SUCCESS",
		"他人のOSSを紹介して、自分だけアクセスや収益を取る。",
		"Other OSS maintainers are encouraged to reuse this policy.",
	)
	requirePolicyFragments(t, "PRESS_KIT.md",
		"Don't build your audience by making the developer disappear.",
		"DoneCanary by @superdoccimo",
		"https://github.com/superdoccimo/done-canary",
		"https://x.com/superdoccimo",
		"他人のOSSを紹介して、自分だけアクセスを取る。",
		"Do not remove `DoneCanary · @superdoccimo` from official share assets.",
	)
}

func TestCreatorFirstAcceptanceIDsAreRecorded(t *testing.T) {
	fragments := make([]string, 0, 13)
	for index := 1; index <= 13; index++ {
		fragments = append(fragments, fmt.Sprintf("AT-STRONG-%03d", index))
	}
	requirePolicyFragments(t, "ACCEPTANCE.md", fragments...)
}
