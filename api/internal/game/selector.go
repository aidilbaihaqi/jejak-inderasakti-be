package game

import (
	"fmt"
	"math/rand/v2"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	stageCount    = 5
	questionsEach = 3
)

// composition holds the difficulty level (1 easy, 2 medium, 3 hard) of the 3 questions per site, per jenjang.
// Source: docs/04_Data_Modelling.md section 8.
var composition = map[string][stageCount][questionsEach]int{
	"SD":  {{1, 1, 2}, {1, 1, 2}, {1, 2, 2}, {1, 2, 2}, {1, 2, 3}},
	"SMP": {{1, 1, 2}, {1, 2, 2}, {1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	"SMA": {{1, 2, 2}, {1, 2, 3}, {1, 2, 3}, {2, 2, 3}, {2, 3, 3}},
}

type siteLevel struct{ site, level int }

// SelectQuestions picks the ordered question ids for a new room: one stage per site,
// random distinct questions per cell. A short session keeps the first and last question of each cell.
func SelectQuestions(rng *rand.Rand, jenjang string, shortSession bool, pool []store.QuestionRef) ([]string, error) {
	cells, ok := composition[jenjang]
	if !ok {
		return nil, fmt.Errorf("unknown jenjang %q", jenjang)
	}
	buckets := shuffledBuckets(rng, pool)
	next := make(map[siteLevel]int)

	var ids []string
	for stage, levels := range cells {
		site := stage + 1
		for _, level := range levelsToAsk(levels, shortSession) {
			key := siteLevel{site, level}
			i := next[key]
			if i >= len(buckets[key]) {
				return nil, fmt.Errorf("not enough questions for site %d level %d", site, level)
			}
			ids = append(ids, buckets[key][i])
			next[key] = i + 1
		}
	}
	return ids, nil
}

func levelsToAsk(levels [questionsEach]int, shortSession bool) []int {
	if shortSession {
		return []int{levels[0], levels[questionsEach-1]}
	}
	return levels[:]
}

func shuffledBuckets(rng *rand.Rand, pool []store.QuestionRef) map[siteLevel][]string {
	buckets := make(map[siteLevel][]string)
	for _, q := range pool {
		key := siteLevel{q.Site, q.Level}
		buckets[key] = append(buckets[key], q.ID)
	}
	for _, ids := range buckets {
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	}
	return buckets
}
