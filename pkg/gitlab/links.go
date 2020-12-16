package gitlab

import "regexp"

var lre = regexp.MustCompile(`<([^>]*)>; *rel="([^"]*)"`)

func parseLinks(h string) map[string]string {
	links := map[string]string{}
	for _, m := range lre.FindAllStringSubmatch(h, 10) {
		if len(m) != 3 {
			continue
		}
		links[m[2]] = m[1]
	}
	return links
}
