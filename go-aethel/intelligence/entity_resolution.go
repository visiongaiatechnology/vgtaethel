package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func (s *Store) ProposeEntityResolution(caseID, leftID, rightID string) (EntityResolutionCandidate, error) {
	caseID, leftID, rightID = strings.TrimSpace(caseID), strings.TrimSpace(leftID), strings.TrimSpace(rightID)
	if caseID == "" || leftID == "" || rightID == "" || leftID == rightID {
		return EntityResolutionCandidate{}, errors.New("resolution requires a case and two distinct entities")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	left, right, found := caseEntityPair(s.state.Cases, caseID, leftID, rightID)
	if !found {
		return EntityResolutionCandidate{}, errors.New("case-local entity pair not found")
	}
	for _, candidate := range s.state.ResolutionCandidates {
		if candidate.CaseID == caseID && unorderedPair(candidate.LeftEntityID, candidate.RightEntityID, leftID, rightID) && candidate.Status == "pending" {
			return candidate, nil
		}
	}
	signals, reasons, score := resolutionScore(left, right)
	id, err := newIntelID("resolution")
	if err != nil {
		return EntityResolutionCandidate{}, err
	}
	candidate := EntityResolutionCandidate{
		ID: id, CaseID: caseID, LeftEntityID: leftID, RightEntityID: rightID,
		Score: score, Signals: signals, Reasons: reasons, Status: "pending", CreatedAt: time.Now().UTC(),
	}
	s.state.ResolutionCandidates = append(s.state.ResolutionCandidates, candidate)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: candidate.CreatedAt, Action: "entity-resolution.proposed", Actor: "resolver", Detail: candidate.ID})
	if err := s.save(); err != nil {
		return EntityResolutionCandidate{}, err
	}
	s.publish("entity-resolution.proposed", candidate.ID, candidate)
	return candidate, nil
}

func (s *Store) ReviewEntityResolution(candidateID, action, actor, reason string) (EntityResolutionDecision, error) {
	candidateID = strings.TrimSpace(candidateID)
	action = strings.ToLower(strings.TrimSpace(action))
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if candidateID == "" || actor == "" || reason == "" {
		return EntityResolutionDecision{}, errors.New("candidate, actor and reason are required")
	}
	if action != "merge" && action != "reject" && action != "split" {
		return EntityResolutionDecision{}, errors.New("resolution action is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.ResolutionCandidates {
		candidate := &s.state.ResolutionCandidates[index]
		if candidate.ID != candidateID {
			continue
		}
		if candidate.Status != "pending" {
			return EntityResolutionDecision{}, errors.New("resolution candidate is already reviewed")
		}
		before := resolutionClusters(s.state.ResolvedEntities, candidate.CaseID)
		switch action {
		case "merge":
			if err := s.mergeResolvedEntitiesLocked(*candidate); err != nil {
				return EntityResolutionDecision{}, err
			}
		case "split":
			if err := s.splitResolvedEntitiesLocked(*candidate); err != nil {
				return EntityResolutionDecision{}, err
			}
		}
		now := time.Now().UTC()
		candidate.Status = map[string]string{"merge": "merged", "reject": "rejected", "split": "split"}[action]
		candidate.ReviewedAt = now
		candidate.ReviewedBy = actor
		decisionID, err := newIntelID("resolution-decision")
		if err != nil {
			return EntityResolutionDecision{}, err
		}
		decision := EntityResolutionDecision{
			ID: decisionID, CandidateID: candidate.ID, CaseID: candidate.CaseID, Action: action,
			Actor: actor, Reason: reason, BeforeClusterIDs: before,
			AfterClusterIDs: resolutionClusters(s.state.ResolvedEntities, candidate.CaseID), At: now,
		}
		s.state.ResolutionDecisions = append(s.state.ResolutionDecisions, decision)
		if action == "merge" || action == "split" {
			if err := s.recordEntityVersionsLocked(candidate.CaseID, candidate.LeftEntityID, candidate.RightEntityID, action, actor, reason, now); err != nil {
				return EntityResolutionDecision{}, err
			}
		}
		s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "entity-resolution." + action, Actor: actor, Detail: decision.ID + " " + reason})
		s.appendCustodyLocked(candidate.ID, "entity-resolution."+action, actor, reason)
		if err := s.save(); err != nil {
			return EntityResolutionDecision{}, err
		}
		s.publish("entity-resolution."+action, candidate.ID, decision)
		return decision, nil
	}
	return EntityResolutionDecision{}, errors.New("resolution candidate not found")
}

func (s *Store) AddResolvedEntityAlias(resolvedID string, alias EntityAlias, actor string) (ResolvedEntity, error) {
	resolvedID, alias.Value, actor = strings.TrimSpace(resolvedID), strings.TrimSpace(alias.Value), strings.TrimSpace(actor)
	if resolvedID == "" || alias.Value == "" || len([]rune(alias.Value)) > 240 || actor == "" {
		return ResolvedEntity{}, errors.New("resolved entity, bounded alias and actor are required")
	}
	if !alias.ValidUntil.IsZero() && !alias.ValidFrom.IsZero() && alias.ValidUntil.Before(alias.ValidFrom) {
		return ResolvedEntity{}, errors.New("alias validity interval is invalid")
	}
	if alias.Script == "" {
		alias.Script = detectScript(alias.Value)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.ResolvedEntities {
		resolved := &s.state.ResolvedEntities[index]
		if resolved.ID != resolvedID {
			continue
		}
		for _, existing := range resolved.Aliases {
			if normalizeEntityName(existing.Value) == normalizeEntityName(alias.Value) && existing.ValidFrom.Equal(alias.ValidFrom) && existing.ValidUntil.Equal(alias.ValidUntil) {
				return *resolved, nil
			}
		}
		resolved.Aliases = append(resolved.Aliases, alias)
		resolved.UpdatedAt = time.Now().UTC()
		if err := s.appendEntityVersionLocked(*resolved, "alias-added", actor, "operator-added alias", resolved.UpdatedAt); err != nil {
			return ResolvedEntity{}, err
		}
		s.state.Audits = append(s.state.Audits, AuditEvent{At: resolved.UpdatedAt, Action: "entity-resolution.alias-added", Actor: actor, Detail: resolved.ID})
		if err := s.save(); err != nil {
			return ResolvedEntity{}, err
		}
		return *resolved, nil
	}
	return ResolvedEntity{}, errors.New("resolved entity not found")
}

func (s *Store) mergeResolvedEntitiesLocked(candidate EntityResolutionCandidate) error {
	left, right, found := caseEntityPair(s.state.Cases, candidate.CaseID, candidate.LeftEntityID, candidate.RightEntityID)
	if !found || left.Kind != right.Kind {
		return errors.New("only case-local entities of the same kind can be merged")
	}
	memberSet := map[string]bool{left.ID: true, right.ID: true}
	aliases := []EntityAlias{{Value: left.Label, Script: detectScript(left.Label)}, {Value: right.Label, Script: detectScript(right.Label)}}
	canonical := left.Label
	kept := make([]ResolvedEntity, 0, len(s.state.ResolvedEntities)+1)
	for _, resolved := range s.state.ResolvedEntities {
		if resolved.CaseID != candidate.CaseID || (!containsString(resolved.SourceEntityIDs, left.ID) && !containsString(resolved.SourceEntityIDs, right.ID)) {
			kept = append(kept, resolved)
			continue
		}
		for _, id := range resolved.SourceEntityIDs {
			memberSet[id] = true
		}
		aliases = append(aliases, resolved.Aliases...)
		if len([]rune(resolved.CanonicalLabel)) > len([]rune(canonical)) {
			canonical = resolved.CanonicalLabel
		}
	}
	members := make([]string, 0, len(memberSet))
	for id := range memberSet {
		members = append(members, id)
	}
	sort.Strings(members)
	id, err := newIntelID("resolved-entity")
	if err != nil {
		return err
	}
	kept = append(kept, ResolvedEntity{ID: id, CaseID: candidate.CaseID, Kind: left.Kind, CanonicalLabel: canonical, SourceEntityIDs: members, Aliases: uniqueAliases(aliases), UpdatedAt: time.Now().UTC()})
	s.state.ResolvedEntities = kept
	return nil
}

func (s *Store) splitResolvedEntitiesLocked(candidate EntityResolutionCandidate) error {
	_, right, found := caseEntityPair(s.state.Cases, candidate.CaseID, candidate.LeftEntityID, candidate.RightEntityID)
	if !found {
		return errors.New("case-local entity pair not found")
	}
	result := make([]ResolvedEntity, 0, len(s.state.ResolvedEntities))
	splitFound := false
	for _, resolved := range s.state.ResolvedEntities {
		if resolved.CaseID != candidate.CaseID || !containsString(resolved.SourceEntityIDs, candidate.LeftEntityID) || !containsString(resolved.SourceEntityIDs, candidate.RightEntityID) {
			result = append(result, resolved)
			continue
		}
		splitFound = true
		members := make([]string, 0, len(resolved.SourceEntityIDs)-1)
		for _, id := range resolved.SourceEntityIDs {
			if id != candidate.RightEntityID {
				members = append(members, id)
			}
		}
		if len(members) > 0 {
			resolved.SourceEntityIDs = members
			resolved.UpdatedAt = time.Now().UTC()
			result = append(result, resolved)
		}
	}
	if !splitFound {
		return errors.New("entities are not members of the same resolved cluster")
	}
	id, err := newIntelID("resolved-entity")
	if err != nil {
		return err
	}
	result = append(result, ResolvedEntity{ID: id, CaseID: candidate.CaseID, Kind: right.Kind, CanonicalLabel: right.Label, SourceEntityIDs: []string{right.ID}, Aliases: []EntityAlias{{Value: right.Label, Script: detectScript(right.Label)}}, UpdatedAt: time.Now().UTC()})
	s.state.ResolvedEntities = result
	return nil
}

func (s *Store) recordEntityVersionsLocked(caseID, leftID, rightID, action, actor, reason string, at time.Time) error {
	for _, resolved := range s.state.ResolvedEntities {
		if resolved.CaseID != caseID || (!containsString(resolved.SourceEntityIDs, leftID) && !containsString(resolved.SourceEntityIDs, rightID)) {
			continue
		}
		if err := s.appendEntityVersionLocked(resolved, action, actor, reason, at); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) appendEntityVersionLocked(snapshot ResolvedEntity, action, actor, reason string, at time.Time) error {
	id, err := newIntelID("entity-version")
	if err != nil {
		return err
	}
	version := 1
	for _, existing := range s.state.EntityVersions {
		if existing.ResolvedEntityID == snapshot.ID && existing.Version >= version {
			version = existing.Version + 1
		}
	}
	s.state.EntityVersions = append(s.state.EntityVersions, EntityVersion{ID: id, ResolvedEntityID: snapshot.ID, CaseID: snapshot.CaseID, Version: version, Action: action, Actor: actor, Reason: reason, Snapshot: snapshot, At: at})
	return nil
}

func resolutionScore(left, right Entity) ([]ResolutionSignal, []string, int) {
	leftNormalized, rightNormalized := normalizeEntityName(left.Label), normalizeEntityName(right.Label)
	dice := bigramDice(leftNormalized, rightNormalized)
	tokens := tokenJaccard(leftNormalized, rightNormalized)
	phonetic := 0.0
	if phoneticKey(leftNormalized) != "" && phoneticKey(leftNormalized) == phoneticKey(rightNormalized) {
		phonetic = 1
	}
	kind := 0.0
	if left.Kind == right.Kind {
		kind = 1
	}
	exact := 0.0
	if leftNormalized != "" && leftNormalized == rightNormalized {
		exact = 1
	}
	signals := []ResolutionSignal{
		{Name: "normalized_exact", Score: exact, Reason: "Unicode-normalized and transliterated labels"},
		{Name: "bigram_similarity", Score: dice, Reason: "Linear-time fuzzy character similarity"},
		{Name: "token_overlap", Score: tokens, Reason: "Order-independent token overlap"},
		{Name: "phonetic_key", Score: phonetic, Reason: "Phonetic consonant signature"},
		{Name: "entity_kind", Score: kind, Reason: "Entity kinds must remain compatible"},
	}
	weighted := exact*0.30 + dice*0.25 + tokens*0.20 + phonetic*0.15 + kind*0.10
	reasons := make([]string, 0, len(signals))
	for _, signal := range signals {
		if signal.Score >= 0.75 {
			reasons = append(reasons, signal.Name+": "+signal.Reason)
		}
	}
	return signals, reasons, int(math.Round(weighted * 100))
}

func normalizeEntityName(value string) string {
	var builder strings.Builder
	for _, r := range norm.NFKD.String(strings.ToLower(strings.TrimSpace(value))) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if mapped, exists := transliterationForRune(r); exists {
			builder.WriteString(mapped)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			builder.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func transliterationForRune(value rune) (string, bool) {
	const cyrillic = "\u0430\u0431\u0432\u0433\u0434\u0435\u0451\u0436\u0437\u0438\u0439\u043a\u043b\u043c\u043d\u043e\u043f\u0440\u0441\u0442\u0443\u0444\u0445\u0446\u0447\u0448\u0449\u044b\u044d\u044e\u044f\u044c\u044a"
	cyrillicASCII := [...]string{"a", "b", "v", "g", "d", "e", "e", "zh", "z", "i", "i", "k", "l", "m", "n", "o", "p", "r", "s", "t", "u", "f", "kh", "ts", "ch", "sh", "shch", "y", "e", "yu", "ya", "", ""}
	index := 0
	for _, candidate := range cyrillic {
		if value == candidate {
			return cyrillicASCII[index], true
		}
		index++
	}
	const greek = "\u03b1\u03b2\u03b3\u03b4\u03b5\u03b6\u03b7\u03b8\u03b9\u03ba\u03bb\u03bc\u03bd\u03be\u03bf\u03c0\u03c1\u03c3\u03c2\u03c4\u03c5\u03c6\u03c7\u03c8\u03c9"
	greekASCII := [...]string{"a", "v", "g", "d", "e", "z", "i", "th", "i", "k", "l", "m", "n", "x", "o", "p", "r", "s", "s", "t", "y", "f", "ch", "ps", "o"}
	index = 0
	for _, candidate := range greek {
		if value == candidate {
			return greekASCII[index], true
		}
		index++
	}
	return "", false
}

var entityTransliteration = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch", 'ы': "y", 'э': "e", 'ю': "yu", 'я': "ya", 'ь': "", 'ъ': "",
	'α': "a", 'β': "v", 'γ': "g", 'δ': "d", 'ε': "e", 'ζ': "z", 'η': "i", 'θ': "th", 'ι': "i", 'κ': "k", 'λ': "l", 'μ': "m", 'ν': "n", 'ξ': "x", 'ο': "o", 'π': "p", 'ρ': "r", 'σ': "s", 'ς': "s", 'τ': "t", 'υ': "y", 'φ': "f", 'χ': "ch", 'ψ': "ps", 'ω': "o",
}

func bigramDice(left, right string) float64 {
	leftRunes, rightRunes := []rune(left), []rune(right)
	if left == right && left != "" {
		return 1
	}
	if len(leftRunes) < 2 || len(rightRunes) < 2 {
		return 0
	}
	counts := make(map[string]int, len(leftRunes)-1)
	for index := 0; index < len(leftRunes)-1; index++ {
		counts[string(leftRunes[index:index+2])]++
	}
	matches := 0
	for index := 0; index < len(rightRunes)-1; index++ {
		key := string(rightRunes[index : index+2])
		if counts[key] > 0 {
			matches++
			counts[key]--
		}
	}
	return float64(2*matches) / float64(len(leftRunes)+len(rightRunes)-2)
}

func tokenJaccard(left, right string) float64 {
	leftSet, rightSet := make(map[string]bool), make(map[string]bool)
	for _, token := range strings.Fields(left) {
		leftSet[token] = true
	}
	for _, token := range strings.Fields(right) {
		rightSet[token] = true
	}
	intersection := 0
	for token := range leftSet {
		if rightSet[token] {
			intersection++
		}
	}
	union := len(leftSet) + len(rightSet) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func phoneticKey(value string) string {
	var builder strings.Builder
	last := byte(0)
	for _, r := range value {
		if r < 'a' || r > 'z' {
			continue
		}
		code := phoneticCode(byte(r))
		if builder.Len() == 0 {
			builder.WriteRune(r)
			last = code
			continue
		}
		if code != 0 && code != last {
			builder.WriteByte(code)
		}
		last = code
	}
	return builder.String()
}

func phoneticCode(r byte) byte {
	switch r {
	case 'b', 'f', 'p', 'v':
		return '1'
	case 'c', 'g', 'j', 'k', 'q', 's', 'x', 'z':
		return '2'
	case 'd', 't':
		return '3'
	case 'l':
		return '4'
	case 'm', 'n':
		return '5'
	case 'r':
		return '6'
	default:
		return 0
	}
}

func caseEntityPair(cases []Case, caseID, leftID, rightID string) (Entity, Entity, bool) {
	var left, right Entity
	leftFound, rightFound := false, false
	for _, caseRecord := range cases {
		if caseRecord.ID != caseID {
			continue
		}
		for _, entity := range caseRecord.Entities {
			if entity.ID == leftID {
				left, leftFound = entity, true
			}
			if entity.ID == rightID {
				right, rightFound = entity, true
			}
		}
	}
	return left, right, leftFound && rightFound
}

func unorderedPair(a, b, c, d string) bool { return (a == c && b == d) || (a == d && b == c) }
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func resolutionClusters(values []ResolvedEntity, caseID string) [][]string {
	result := make([][]string, 0)
	for _, value := range values {
		if value.CaseID == caseID {
			result = append(result, append([]string(nil), value.SourceEntityIDs...))
		}
	}
	return result
}
func uniqueAliases(values []EntityAlias) []EntityAlias {
	seen := make(map[string]bool, len(values))
	result := make([]EntityAlias, 0, len(values))
	for _, value := range values {
		key := normalizeEntityName(value.Value)
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result
}
func detectScript(value string) string {
	for _, r := range value {
		if unicode.In(r, unicode.Cyrillic) {
			return "Cyrillic"
		}
		if unicode.In(r, unicode.Greek) {
			return "Greek"
		}
	}
	return "Latin"
}
