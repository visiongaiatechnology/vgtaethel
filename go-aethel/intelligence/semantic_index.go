package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

const (
	semanticDimensions = 2048
	semanticThreshold  = 0.24
)

type sparseEmbedding map[uint32]float64

type localSemanticIndex struct {
	mu       sync.RWMutex
	revision uint64
	vectors  map[string]sparseEmbedding
}

func newLocalSemanticIndex() *localSemanticIndex {
	return &localSemanticIndex{vectors: make(map[string]sparseEmbedding)}
}

func (index *localSemanticIndex) sync(state StoreState) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.revision == state.Revision && len(index.vectors) > 0 {
		return
	}
	vectors := make(map[string]sparseEmbedding)
	for _, record := range searchableIndexRecords(state) {
		vectors[record.recordType+":"+record.recordID] = embedLocalText(record.content)
	}
	index.vectors, index.revision = vectors, state.Revision
}

func (index *localSemanticIndex) query(text string, limit int) map[string]float64 {
	query := embedLocalText(text)
	index.mu.RLock()
	type scored struct {
		key   string
		score float64
	}
	scores := make([]scored, 0)
	for key, vector := range index.vectors {
		if score := sparseCosine(query, vector); score >= semanticThreshold {
			scores = append(scores, scored{key, score})
		}
	}
	index.mu.RUnlock()
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].key < scores[j].key
		}
		return scores[i].score > scores[j].score
	})
	if limit < 1 {
		limit = 100
	}
	if len(scores) > limit {
		scores = scores[:limit]
	}
	result := make(map[string]float64, len(scores))
	for _, item := range scores {
		result[item.key] = item.score
	}
	return result
}

func embedLocalText(text string) sparseEmbedding {
	runes := []rune(strings.ToLower(strings.TrimSpace(text)))
	if len(runes) > 131072 {
		runes = runes[:131072]
	}
	vector := make(sparseEmbedding)
	word := make([]rune, 0, 32)
	flushWord := func() {
		if len(word) == 0 {
			return
		}
		addEmbeddingFeature(vector, "w:"+string(word), 1.6)
		word = word[:0]
	}
	for _, current := range runes {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			word = append(word, current)
		} else {
			flushWord()
		}
	}
	flushWord()
	compact := make([]rune, 0, len(runes))
	for _, current := range runes {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			compact = append(compact, current)
		}
	}
	for width := 2; width <= 4; width++ {
		for start := 0; start+width <= len(compact); start++ {
			addEmbeddingFeature(vector, "g:"+string(compact[start:start+width]), 1)
		}
	}
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for key, value := range vector {
			vector[key] = value / norm
		}
	}
	return vector
}

func addEmbeddingFeature(vector sparseEmbedding, feature string, weight float64) {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(feature))
	key := uint32(hasher.Sum64() % semanticDimensions)
	vector[key] += weight
}

func sparseCosine(left, right sparseEmbedding) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	if len(left) > len(right) {
		left, right = right, left
	}
	score := 0.0
	for key, value := range left {
		score += value * right[key]
	}
	return score
}
