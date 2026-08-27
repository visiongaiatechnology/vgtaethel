package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math/bits"
	"sort"
	"strings"
	"time"
)

const (
	maximumImageBytes  = 8 << 20
	maximumImagePixels = 40_000_000
)

type ImageFingerprint struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"case_id,omitempty"`
	SourceID   string    `json:"source_id,omitempty"`
	Label      string    `json:"label"`
	SHA256     string    `json:"sha256"`
	Format     string    `json:"format"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Difference uint64    `json:"difference_hash"`
	CreatedAt  time.Time `json:"created_at"`
}

type ImageMatch struct {
	Fingerprint ImageFingerprint `json:"fingerprint"`
	Distance    int              `json:"hamming_distance"`
	Similarity  int              `json:"similarity_percent"`
}

func (s *Store) MatchImage(raw []byte, caseID, sourceID, label string, index bool) (ImageFingerprint, []ImageMatch, error) {
	fingerprint, err := buildImageFingerprint(raw, caseID, sourceID, label)
	if err != nil {
		return ImageFingerprint{}, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	matches := make([]ImageMatch, 0, len(s.state.ImageFingerprints))
	for _, candidate := range s.state.ImageFingerprints {
		distance := bits.OnesCount64(fingerprint.Difference ^ candidate.Difference)
		matches = append(matches, ImageMatch{Fingerprint: candidate, Distance: distance, Similarity: (64 - distance) * 100 / 64})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Distance == matches[j].Distance {
			return matches[i].Fingerprint.CreatedAt.After(matches[j].Fingerprint.CreatedAt)
		}
		return matches[i].Distance < matches[j].Distance
	})
	if len(matches) > 50 {
		matches = matches[:50]
	}
	if index {
		for _, existing := range s.state.ImageFingerprints {
			if existing.SHA256 == fingerprint.SHA256 && existing.CaseID == fingerprint.CaseID {
				return existing, matches, nil
			}
		}
		s.state.ImageFingerprints = append(s.state.ImageFingerprints, fingerprint)
		s.state.Audits = append(s.state.Audits, AuditEvent{At: fingerprint.CreatedAt, Action: "image-fingerprint.indexed", Actor: "operator", Detail: fingerprint.ID})
		if err := s.save(); err != nil {
			return ImageFingerprint{}, nil, err
		}
	}
	return fingerprint, matches, nil
}

func buildImageFingerprint(raw []byte, caseID, sourceID, label string) (ImageFingerprint, error) {
	if len(raw) == 0 || len(raw) > maximumImageBytes {
		return ImageFingerprint{}, errors.New("image size boundary violation")
	}
	caseID, sourceID, label = strings.TrimSpace(caseID), strings.TrimSpace(sourceID), strings.TrimSpace(label)
	if len([]rune(caseID)) > 160 || len([]rune(sourceID)) > 160 || label == "" || len([]rune(label)) > 240 {
		return ImageFingerprint{}, errors.New("image metadata is invalid")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 20_000 || config.Height > 20_000 || int64(config.Width)*int64(config.Height) > maximumImagePixels {
		return ImageFingerprint{}, errors.New("unsupported or unsafe image geometry")
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(raw))
	if err != nil || decodedFormat != format {
		return ImageFingerprint{}, errors.New("image decoding failed")
	}
	id, err := newIntelID("image")
	if err != nil {
		return ImageFingerprint{}, err
	}
	digest := sha256.Sum256(raw)
	return ImageFingerprint{
		ID: id, CaseID: caseID, SourceID: sourceID, Label: label,
		SHA256: hex.EncodeToString(digest[:]), Format: format, Width: config.Width, Height: config.Height,
		Difference: differenceHash(decoded), CreatedAt: time.Now().UTC(),
	}, nil
}

func differenceHash(source image.Image) uint64 {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	var luminance [72]uint32
	for y := 0; y < 8; y++ {
		for x := 0; x < 9; x++ {
			sourceX := bounds.Min.X + ((2*x+1)*width)/(2*9)
			sourceY := bounds.Min.Y + ((2*y+1)*height)/(2*8)
			if sourceX >= bounds.Max.X {
				sourceX = bounds.Max.X - 1
			}
			if sourceY >= bounds.Max.Y {
				sourceY = bounds.Max.Y - 1
			}
			r, g, b, _ := source.At(sourceX, sourceY).RGBA()
			luminance[y*9+x] = (299*r + 587*g + 114*b) / 1000
		}
	}
	var hash uint64
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if luminance[y*9+x] > luminance[y*9+x+1] {
				hash |= uint64(1) << uint(y*8+x)
			}
		}
	}
	return hash
}
