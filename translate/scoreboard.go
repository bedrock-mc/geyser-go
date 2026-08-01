package translate

import (
	"fmt"
	"sort"
	"strings"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaScoreboardObjectiveAdd    int8 = 0
	javaScoreboardObjectiveRemove int8 = 1
	javaScoreboardObjectiveUpdate int8 = 2

	javaScoreboardTeamCreate        int8 = 0
	javaScoreboardTeamRemove        int8 = 1
	javaScoreboardTeamUpdate        int8 = 2
	javaScoreboardTeamAddPlayers    int8 = 3
	javaScoreboardTeamRemovePlayers int8 = 4

	maxBedrockScoreboardText = 256
	maxBedrockSidebarEntries = 15
)

// JavaScoreboardObjectiveUpdate is the wire shape of the Java
// ClientboundSetObjectivePacket. The number-format style is decoded and
// consumed even though Bedrock's scoreboard packet has no matching field.
type JavaScoreboardObjectiveUpdate struct {
	Name         string
	Action       int8
	DisplayName  any
	RenderType   int32
	NumberFormat *int32
}

// JavaScoreboardDisplayObjective is the wire shape of the Java display-slot
// packet. Java has coloured sidebar slots; Bedrock exposes one sidebar slot,
// so all Java sidebar variants intentionally map to it.
type JavaScoreboardDisplayObjective struct {
	Position int32
	Name     string
}

// JavaScoreboardScore is the wire shape of one Java score update.
type JavaScoreboardScore struct {
	Owner          string
	Objective      string
	Value          int32
	DisplayName    any
	HasDisplayName bool
	NumberFormat   *int32
}

// JavaResetScore is the wire shape of the Java reset-score packet.
type JavaResetScore struct {
	Owner     string
	Objective *string
}

// JavaScoreboardTeam is the wire shape of the Java team packet. Bedrock does
// not expose Java's visibility/collision/color controls in SetScore, but the
// prefix and suffix are projected into the visible fake-player line.
type JavaScoreboardTeam struct {
	Name              string
	Mode              int8
	DisplayName       any
	FriendlyFire      int8
	NameTagVisibility string
	CollisionRule     string
	Formatting        int32
	Prefix            any
	Suffix            any
	Players           []string
}

func decodeJavaOptionalNBT(r *javaprotocol.Reader, field string) (any, bool, error) {
	present, err := r.Bool()
	if err != nil {
		return nil, false, fmt.Errorf("translate: %s presence: %w", field, err)
	}
	if !present {
		return nil, false, nil
	}
	value, err := decodeJavaNBTValue(r)
	if err != nil {
		return nil, false, fmt.Errorf("translate: %s: %w", field, err)
	}
	return value, true, nil
}

// decodeJavaNumberFormat consumes the Java optional number-format holder and
// its optional styling component. Format IDs 1 and 2 carry an anonymous NBT
// style; all other IDs are self-contained and are retained as opaque state.
func decodeJavaNumberFormat(r *javaprotocol.Reader, field string) (*int32, error) {
	present, err := r.Bool()
	if err != nil {
		return nil, fmt.Errorf("translate: %s presence: %w", field, err)
	}
	if !present {
		return nil, nil
	}
	id, err := r.VarInt()
	if err != nil {
		return nil, fmt.Errorf("translate: %s ID: %w", field, err)
	}
	if id == 1 || id == 2 {
		if _, err := decodeJavaNBTValue(r); err != nil {
			return nil, fmt.Errorf("translate: %s style: %w", field, err)
		}
	}
	return &id, nil
}

func DecodeJavaScoreboardObjective(data []byte) (JavaScoreboardObjectiveUpdate, error) {
	r := javaprotocol.NewReader(data)
	name, err := r.String()
	if err != nil {
		return JavaScoreboardObjectiveUpdate{}, fmt.Errorf("translate: scoreboard objective name: %w", err)
	}
	action, err := r.Int8()
	if err != nil {
		return JavaScoreboardObjectiveUpdate{}, fmt.Errorf("translate: scoreboard objective action: %w", err)
	}
	update := JavaScoreboardObjectiveUpdate{Name: name, Action: action}
	if action == javaScoreboardObjectiveAdd || action == javaScoreboardObjectiveUpdate {
		if update.DisplayName, err = decodeJavaNBTValue(r); err != nil {
			return JavaScoreboardObjectiveUpdate{}, fmt.Errorf("translate: scoreboard objective display name: %w", err)
		}
		if update.RenderType, err = r.VarInt(); err != nil {
			return JavaScoreboardObjectiveUpdate{}, fmt.Errorf("translate: scoreboard objective render type: %w", err)
		}
		if update.NumberFormat, err = decodeJavaNumberFormat(r, "scoreboard objective number format"); err != nil {
			return JavaScoreboardObjectiveUpdate{}, err
		}
	}
	if r.Remaining() != 0 {
		return JavaScoreboardObjectiveUpdate{}, fmt.Errorf("translate: scoreboard objective has %d trailing bytes", r.Remaining())
	}
	return update, nil
}

func DecodeJavaScoreboardDisplayObjective(data []byte) (JavaScoreboardDisplayObjective, error) {
	r := javaprotocol.NewReader(data)
	position, err := r.VarInt()
	if err != nil {
		return JavaScoreboardDisplayObjective{}, fmt.Errorf("translate: scoreboard display position: %w", err)
	}
	name, err := r.String()
	if err != nil {
		return JavaScoreboardDisplayObjective{}, fmt.Errorf("translate: scoreboard display name: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaScoreboardDisplayObjective{}, fmt.Errorf("translate: scoreboard display has %d trailing bytes", r.Remaining())
	}
	return JavaScoreboardDisplayObjective{Position: position, Name: name}, nil
}

func DecodeJavaScoreboardScore(data []byte) (JavaScoreboardScore, error) {
	r := javaprotocol.NewReader(data)
	owner, err := r.String()
	if err != nil {
		return JavaScoreboardScore{}, fmt.Errorf("translate: scoreboard score owner: %w", err)
	}
	objective, err := r.String()
	if err != nil {
		return JavaScoreboardScore{}, fmt.Errorf("translate: scoreboard score objective: %w", err)
	}
	value, err := r.VarInt()
	if err != nil {
		return JavaScoreboardScore{}, fmt.Errorf("translate: scoreboard score value: %w", err)
	}
	display, present, err := decodeJavaOptionalNBT(r, "scoreboard score display name")
	if err != nil {
		return JavaScoreboardScore{}, err
	}
	numberFormat, err := decodeJavaNumberFormat(r, "scoreboard score number format")
	if err != nil {
		return JavaScoreboardScore{}, err
	}
	if r.Remaining() != 0 {
		return JavaScoreboardScore{}, fmt.Errorf("translate: scoreboard score has %d trailing bytes", r.Remaining())
	}
	return JavaScoreboardScore{
		Owner: owner, Objective: objective, Value: value,
		DisplayName: display, HasDisplayName: present, NumberFormat: numberFormat,
	}, nil
}

func DecodeJavaResetScore(data []byte) (JavaResetScore, error) {
	r := javaprotocol.NewReader(data)
	owner, err := r.String()
	if err != nil {
		return JavaResetScore{}, fmt.Errorf("translate: reset score owner: %w", err)
	}
	objective, present, err := decodeJavaOptionalString(r, "reset score objective")
	if err != nil {
		return JavaResetScore{}, err
	}
	if r.Remaining() != 0 {
		return JavaResetScore{}, fmt.Errorf("translate: reset score has %d trailing bytes", r.Remaining())
	}
	if !present {
		return JavaResetScore{Owner: owner}, nil
	}
	return JavaResetScore{Owner: owner, Objective: &objective}, nil
}

func decodeJavaOptionalString(r *javaprotocol.Reader, field string) (string, bool, error) {
	present, err := r.Bool()
	if err != nil {
		return "", false, fmt.Errorf("translate: %s presence: %w", field, err)
	}
	if !present {
		return "", false, nil
	}
	value, err := r.String()
	if err != nil {
		return "", false, fmt.Errorf("translate: %s: %w", field, err)
	}
	return value, true, nil
}

func DecodeJavaScoreboardTeam(data []byte) (JavaScoreboardTeam, error) {
	r := javaprotocol.NewReader(data)
	name, err := r.String()
	if err != nil {
		return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team name: %w", err)
	}
	mode, err := r.Int8()
	if err != nil {
		return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team mode: %w", err)
	}
	team := JavaScoreboardTeam{Name: name, Mode: mode}
	if mode == javaScoreboardTeamCreate || mode == javaScoreboardTeamUpdate {
		if team.DisplayName, err = decodeJavaNBTValue(r); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team display name: %w", err)
		}
		if team.FriendlyFire, err = r.Int8(); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team friendly-fire: %w", err)
		}
		if team.NameTagVisibility, err = r.String(); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team name-tag visibility: %w", err)
		}
		if team.CollisionRule, err = r.String(); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team collision rule: %w", err)
		}
		if team.Formatting, err = r.VarInt(); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team formatting: %w", err)
		}
		if team.Prefix, err = decodeJavaNBTValue(r); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team prefix: %w", err)
		}
		if team.Suffix, err = decodeJavaNBTValue(r); err != nil {
			return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team suffix: %w", err)
		}
	}
	if mode == javaScoreboardTeamCreate || mode == javaScoreboardTeamAddPlayers || mode == javaScoreboardTeamRemovePlayers {
		count, err := boundedJavaCount(r, "scoreboard team player count")
		if err != nil {
			return JavaScoreboardTeam{}, err
		}
		team.Players = make([]string, count)
		for i := range team.Players {
			if team.Players[i], err = r.String(); err != nil {
				return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team player %d: %w", i, err)
			}
		}
	}
	if r.Remaining() != 0 {
		return JavaScoreboardTeam{}, fmt.Errorf("translate: scoreboard team has %d trailing bytes", r.Remaining())
	}
	return team, nil
}

type javaScoreboardObjectiveState struct {
	Name        string
	DisplayName string
	Scores      map[string]*javaScoreboardScoreState
	Active      map[int32]*javaScoreboardDisplayState
}

type javaScoreboardScoreState struct {
	Owner       string
	Value       int32
	DisplayName string
}

type javaScoreboardDisplayState struct {
	BedrockID string
	Slot      string
	Rendered  map[string]gtprotocol.ScoreboardEntry
}

type javaScoreboardTeamState struct {
	Name    string
	Players map[string]struct{}
	Prefix  string
	Suffix  string
}

func scoreboardSlot(position int32) (string, bool) {
	switch position {
	case 0:
		return packet.ScoreboardSlotList, true
	case 1:
		return packet.ScoreboardSlotSidebar, true
	case 2:
		return packet.ScoreboardSlotBelowName, true
	case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18:
		return packet.ScoreboardSlotSidebar, true
	default:
		return "", false
	}
}

func limitScoreboardText(value string) string {
	runes := []rune(value)
	if len(runes) <= maxBedrockScoreboardText {
		return value
	}
	return string(runes[:maxBedrockScoreboardText])
}

func (b *Basic) newScoreboardObjectiveIDLocked() string {
	id := b.nextScoreboardID
	if id <= 0 {
		id = 1
	}
	b.nextScoreboardID = id + 1
	if b.nextScoreboardID <= 0 {
		b.nextScoreboardID = 1
	}
	return fmt.Sprintf("geyser-go-score-%d", id)
}

func (b *Basic) newScoreboardEntryIDLocked() int64 {
	id := b.nextScoreboardEntryID
	if id <= 0 {
		id = 1
	}
	b.nextScoreboardEntryID = id + 1
	if b.nextScoreboardEntryID <= 0 {
		b.nextScoreboardEntryID = 1
	}
	return id
}

func (b *Basic) scoreboardTeamForOwnerLocked(owner string) *javaScoreboardTeamState {
	for _, team := range b.scoreboardTeams {
		if _, ok := team.Players[owner]; ok {
			return team
		}
	}
	return nil
}

func (b *Basic) addScoreboardTeamPlayersLocked(team *javaScoreboardTeamState, players []string) {
	for _, player := range players {
		for _, other := range b.scoreboardTeams {
			if other != team {
				delete(other.Players, player)
			}
		}
		team.Players[player] = struct{}{}
	}
}

func (b *Basic) scoreboardEntryDisplayNameLocked(score *javaScoreboardScoreState) string {
	name := score.DisplayName
	if name == "" {
		name = score.Owner
	}
	if team := b.scoreboardTeamForOwnerLocked(score.Owner); team != nil {
		name = team.Prefix + name + team.Suffix
	}
	return limitScoreboardText(name)
}

func (b *Basic) buildScoreboardEntriesLocked(objective *javaScoreboardObjectiveState, display *javaScoreboardDisplayState) []gtprotocol.ScoreboardEntry {
	previous := display.Rendered
	scores := make([]*javaScoreboardScoreState, 0, len(objective.Scores))
	for _, score := range objective.Scores {
		scores = append(scores, score)
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Value != scores[j].Value {
			return scores[i].Value > scores[j].Value
		}
		left, right := strings.ToLower(scores[i].Owner), strings.ToLower(scores[j].Owner)
		if left != right {
			return left < right
		}
		return scores[i].Owner < scores[j].Owner
	})
	if display.Slot == packet.ScoreboardSlotSidebar && len(scores) > maxBedrockSidebarEntries {
		scores = scores[:maxBedrockSidebarEntries]
	}
	entries := make([]gtprotocol.ScoreboardEntry, 0, len(scores))
	rendered := make(map[string]gtprotocol.ScoreboardEntry, len(scores))
	for _, score := range scores {
		entry, ok := previous[score.Owner]
		if !ok {
			entry = gtprotocol.ScoreboardEntry{
				EntryID:       b.newScoreboardEntryIDLocked(),
				ObjectiveName: display.BedrockID,
				IdentityType:  gtprotocol.ScoreboardIdentityFakePlayer,
			}
		}
		entry.ObjectiveName = display.BedrockID
		entry.Score = score.Value
		entry.IdentityType = gtprotocol.ScoreboardIdentityFakePlayer
		entry.DisplayName = b.scoreboardEntryDisplayNameLocked(score)
		entries = append(entries, entry)
		rendered[score.Owner] = entry
	}
	display.Rendered = rendered
	return entries
}

func (b *Basic) renderScoreboardEntries(bedrock *minecraft.Conn, objectiveName string, position int32) error {
	b.mu.Lock()
	objective := b.scoreboardObjectives[objectiveName]
	if objective == nil {
		b.mu.Unlock()
		return nil
	}
	display := objective.Active[position]
	if display == nil {
		b.mu.Unlock()
		return nil
	}
	old := make([]gtprotocol.ScoreboardEntry, 0, len(display.Rendered))
	for _, entry := range display.Rendered {
		old = append(old, entry)
	}
	entries := b.buildScoreboardEntriesLocked(objective, display)
	b.mu.Unlock()

	if len(old) > 0 {
		if err := bedrock.WritePacket(&packet.SetScore{ActionType: packet.ScoreboardActionRemove, Entries: old}); err != nil {
			return err
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return bedrock.WritePacket(&packet.SetScore{ActionType: packet.ScoreboardActionModify, Entries: entries})
}

func (b *Basic) refreshScoreboardDisplay(bedrock *minecraft.Conn, objectiveName string, position int32, replaceObjective bool) error {
	b.mu.Lock()
	objective := b.scoreboardObjectives[objectiveName]
	if objective == nil || objective.Active[position] == nil {
		b.mu.Unlock()
		return nil
	}
	display := objective.Active[position]
	title := objective.DisplayName
	if title == "" {
		title = " "
	}
	title = limitScoreboardText(title)
	if replaceObjective {
		display.Rendered = nil
	}
	bedrockID, slot := display.BedrockID, display.Slot
	b.mu.Unlock()

	if replaceObjective {
		if err := bedrock.WritePacket(&packet.RemoveObjective{ObjectiveName: bedrockID}); err != nil {
			return err
		}
		if err := bedrock.WritePacket(&packet.SetDisplayObjective{
			DisplaySlot: slot, ObjectiveName: bedrockID, DisplayName: title,
			CriteriaName: "dummy", SortOrder: packet.ScoreboardSortOrderDescending,
		}); err != nil {
			return err
		}
	}
	return b.renderScoreboardEntries(bedrock, objectiveName, position)
}

func (b *Basic) refreshAllScoreboards(bedrock *minecraft.Conn) error {
	b.mu.Lock()
	active := make([]struct {
		objective string
		position  int32
	}, 0)
	for name, objective := range b.scoreboardObjectives {
		for position := range objective.Active {
			active = append(active, struct {
				objective string
				position  int32
			}{name, position})
		}
	}
	b.mu.Unlock()
	sort.Slice(active, func(i, j int) bool {
		if active[i].objective != active[j].objective {
			return active[i].objective < active[j].objective
		}
		return active[i].position < active[j].position
	})
	for _, display := range active {
		if err := b.renderScoreboardEntries(bedrock, display.objective, display.position); err != nil {
			return err
		}
	}
	return nil
}

func (b *Basic) translateJavaScoreboardObjective(bedrock *minecraft.Conn, data []byte) error {
	update, err := DecodeJavaScoreboardObjective(data)
	if err != nil {
		return err
	}
	switch update.Action {
	case javaScoreboardObjectiveRemove:
		b.mu.Lock()
		objective := b.scoreboardObjectives[update.Name]
		var ids []string
		if objective != nil {
			for _, display := range objective.Active {
				ids = append(ids, display.BedrockID)
			}
			delete(b.scoreboardObjectives, update.Name)
		}
		b.mu.Unlock()
		for _, id := range ids {
			if err := bedrock.WritePacket(&packet.RemoveObjective{ObjectiveName: id}); err != nil {
				return err
			}
		}
		return nil
	case javaScoreboardObjectiveAdd, javaScoreboardObjectiveUpdate:
		b.mu.Lock()
		objective := b.scoreboardObjectives[update.Name]
		if objective == nil {
			objective = &javaScoreboardObjectiveState{
				Name: update.Name, Scores: make(map[string]*javaScoreboardScoreState),
				Active: make(map[int32]*javaScoreboardDisplayState),
			}
			b.scoreboardObjectives[update.Name] = objective
		}
		objective.DisplayName = JavaTextComponentText(update.DisplayName)
		positions := make([]int32, 0, len(objective.Active))
		for position := range objective.Active {
			positions = append(positions, position)
		}
		b.mu.Unlock()
		sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
		for _, position := range positions {
			if err := b.refreshScoreboardDisplay(bedrock, update.Name, position, true); err != nil {
				return err
			}
		}
		return nil
	default:
		b.logSemanticAnomaly("skipping Java scoreboard objective with unknown action", "action", update.Action)
		return nil
	}
}

func (b *Basic) translateJavaScoreboardDisplay(bedrock *minecraft.Conn, data []byte) error {
	display, err := DecodeJavaScoreboardDisplayObjective(data)
	if err != nil {
		return err
	}
	slot, ok := scoreboardSlot(display.Position)
	if !ok {
		b.logSemanticAnomaly("skipping Java scoreboard display slot outside Bedrock slots", "position", display.Position)
		return nil
	}
	b.mu.Lock()
	if display.Name == "" {
		var oldIDs []string
		for _, objective := range b.scoreboardObjectives {
			if current := objective.Active[display.Position]; current != nil {
				delete(objective.Active, display.Position)
				oldIDs = append(oldIDs, current.BedrockID)
			}
		}
		b.mu.Unlock()
		for _, id := range oldIDs {
			if err := bedrock.WritePacket(&packet.RemoveObjective{ObjectiveName: id}); err != nil {
				return err
			}
		}
		return nil
	}
	objective := b.scoreboardObjectives[display.Name]
	if objective == nil && display.Name != "" {
		b.logSemanticAnomaly("Java scoreboard display referenced an unknown objective", "objective", display.Name)
		objective = &javaScoreboardObjectiveState{
			Name: display.Name, DisplayName: display.Name,
			Scores: make(map[string]*javaScoreboardScoreState), Active: make(map[int32]*javaScoreboardDisplayState),
		}
		b.scoreboardObjectives[display.Name] = objective
	}
	old := objective.Active[display.Position]
	if old != nil {
		delete(objective.Active, display.Position)
	}
	newDisplay := &javaScoreboardDisplayState{
		BedrockID: b.newScoreboardObjectiveIDLocked(), Slot: slot,
	}
	objective.Active[display.Position] = newDisplay
	title := objective.DisplayName
	if title == "" {
		title = " "
	}
	title = limitScoreboardText(title)
	b.mu.Unlock()
	if old != nil {
		if err := bedrock.WritePacket(&packet.RemoveObjective{ObjectiveName: old.BedrockID}); err != nil {
			return err
		}
	}
	if err := bedrock.WritePacket(&packet.SetDisplayObjective{
		DisplaySlot: slot, ObjectiveName: newDisplay.BedrockID, DisplayName: title,
		CriteriaName: "dummy", SortOrder: packet.ScoreboardSortOrderDescending,
	}); err != nil {
		return err
	}
	return b.renderScoreboardEntries(bedrock, display.Name, display.Position)
}

func (b *Basic) translateJavaScoreboardScore(bedrock *minecraft.Conn, data []byte) error {
	score, err := DecodeJavaScoreboardScore(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	objective := b.scoreboardObjectives[score.Objective]
	if objective == nil {
		b.logSemanticAnomaly("Java scoreboard score referenced an unknown objective", "objective", score.Objective)
		objective = &javaScoreboardObjectiveState{
			Name: score.Objective, Scores: make(map[string]*javaScoreboardScoreState),
			Active: make(map[int32]*javaScoreboardDisplayState),
		}
		b.scoreboardObjectives[score.Objective] = objective
	}
	value := score.Owner
	if !score.HasDisplayName {
		value = ""
	} else {
		value = JavaTextComponentText(score.DisplayName)
	}
	objective.Scores[score.Owner] = &javaScoreboardScoreState{Owner: score.Owner, Value: score.Value, DisplayName: value}
	positions := make([]int32, 0, len(objective.Active))
	for position := range objective.Active {
		positions = append(positions, position)
	}
	b.mu.Unlock()
	sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
	for _, position := range positions {
		if err := b.renderScoreboardEntries(bedrock, score.Objective, position); err != nil {
			return err
		}
	}
	return nil
}

func (b *Basic) translateJavaResetScore(bedrock *minecraft.Conn, data []byte) error {
	reset, err := DecodeJavaResetScore(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	names := make([]string, 0, 1)
	if reset.Objective != nil {
		names = append(names, *reset.Objective)
	} else {
		for name := range b.scoreboardObjectives {
			names = append(names, name)
		}
	}
	positions := make([]struct {
		objective string
		position  int32
	}, 0)
	for _, name := range names {
		if objective := b.scoreboardObjectives[name]; objective != nil {
			delete(objective.Scores, reset.Owner)
			for position := range objective.Active {
				positions = append(positions, struct {
					objective string
					position  int32
				}{name, position})
			}
		}
	}
	b.mu.Unlock()
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].objective != positions[j].objective {
			return positions[i].objective < positions[j].objective
		}
		return positions[i].position < positions[j].position
	})
	for _, position := range positions {
		if err := b.renderScoreboardEntries(bedrock, position.objective, position.position); err != nil {
			return err
		}
	}
	return nil
}

func (b *Basic) translateJavaScoreboardTeam(bedrock *minecraft.Conn, data []byte) error {
	team, err := DecodeJavaScoreboardTeam(data)
	if err != nil {
		return err
	}
	b.mu.Lock()
	state := b.scoreboardTeams[team.Name]
	switch team.Mode {
	case javaScoreboardTeamCreate:
		if state != nil {
			state.Players = make(map[string]struct{})
		} else {
			state = &javaScoreboardTeamState{Name: team.Name, Players: make(map[string]struct{})}
			b.scoreboardTeams[team.Name] = state
		}
		state.Prefix = JavaTextComponentText(team.Prefix)
		state.Suffix = JavaTextComponentText(team.Suffix)
		b.addScoreboardTeamPlayersLocked(state, team.Players)
	case javaScoreboardTeamUpdate:
		if state == nil {
			state = &javaScoreboardTeamState{Name: team.Name, Players: make(map[string]struct{})}
			b.scoreboardTeams[team.Name] = state
		}
		state.Prefix = JavaTextComponentText(team.Prefix)
		state.Suffix = JavaTextComponentText(team.Suffix)
	case javaScoreboardTeamAddPlayers:
		if state == nil {
			state = &javaScoreboardTeamState{Name: team.Name, Players: make(map[string]struct{})}
			b.scoreboardTeams[team.Name] = state
		}
		b.addScoreboardTeamPlayersLocked(state, team.Players)
	case javaScoreboardTeamRemovePlayers:
		if state != nil {
			for _, player := range team.Players {
				delete(state.Players, player)
			}
		}
	case javaScoreboardTeamRemove:
		delete(b.scoreboardTeams, team.Name)
	default:
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping Java scoreboard team with unknown mode", "mode", team.Mode)
		return nil
	}
	b.mu.Unlock()
	return b.refreshAllScoreboards(bedrock)
}
