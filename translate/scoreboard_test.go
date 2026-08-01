package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func writeAnonymousNBTString(t *testing.T, w *javaprotocol.Writer, value string) {
	t.Helper()
	if err := w.Byte(8); err != nil {
		t.Fatal(err)
	}
	if err := w.Int16(int16(len([]byte(value)))); err != nil {
		t.Fatal(err)
	}
	if err := w.BytesValue([]byte(value)); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeJavaScoreboardPackets(t *testing.T) {
	objectiveWriter := javaprotocol.NewWriter()
	_ = objectiveWriter.String("points")
	_ = objectiveWriter.Byte(byte(javaScoreboardObjectiveAdd))
	writeAnonymousNBTString(t, objectiveWriter, "Points")
	_ = objectiveWriter.VarInt(0)
	_ = objectiveWriter.Bool(false)
	objective, err := DecodeJavaScoreboardObjective(objectiveWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if objective.Name != "points" || objective.Action != javaScoreboardObjectiveAdd || JavaTextComponentText(objective.DisplayName) != "Points" {
		t.Fatalf("objective = %+v", objective)
	}

	scoreWriter := javaprotocol.NewWriter()
	_ = scoreWriter.String("First line")
	_ = scoreWriter.String("points")
	_ = scoreWriter.VarInt(7)
	_ = scoreWriter.Bool(true)
	writeAnonymousNBTString(t, scoreWriter, "Visible")
	_ = scoreWriter.Bool(false)
	score, err := DecodeJavaScoreboardScore(scoreWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if score.Owner != "First line" || score.Objective != "points" || score.Value != 7 || !score.HasDisplayName || JavaTextComponentText(score.DisplayName) != "Visible" {
		t.Fatalf("score = %+v", score)
	}

	teamWriter := javaprotocol.NewWriter()
	_ = teamWriter.String("decor")
	_ = teamWriter.Byte(byte(javaScoreboardTeamCreate))
	writeAnonymousNBTString(t, teamWriter, "Decor")
	_ = teamWriter.Byte(0)
	_ = teamWriter.String("always")
	_ = teamWriter.String("always")
	_ = teamWriter.VarInt(-1)
	writeAnonymousNBTString(t, teamWriter, "[P] ")
	writeAnonymousNBTString(t, teamWriter, "!")
	_ = teamWriter.VarInt(1)
	_ = teamWriter.String("First line")
	team, err := DecodeJavaScoreboardTeam(teamWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if team.Name != "decor" || team.Mode != javaScoreboardTeamCreate || JavaTextComponentText(team.Prefix) != "[P] " || JavaTextComponentText(team.Suffix) != "!" || len(team.Players) != 1 {
		t.Fatalf("team = %+v", team)
	}

	resetWriter := javaprotocol.NewWriter()
	_ = resetWriter.String("First line")
	_ = resetWriter.Bool(false)
	reset, err := DecodeJavaResetScore(resetWriter.Bytes())
	if err != nil || reset.Owner != "First line" || reset.Objective != nil {
		t.Fatalf("reset = %+v, err=%v", reset, err)
	}
}

func TestScoreboardProjectionState(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	b.scoreboardTeams["decor"] = &javaScoreboardTeamState{
		Name: "decor", Prefix: "[P] ", Suffix: "!", Players: map[string]struct{}{"alpha": {}},
	}
	objective := &javaScoreboardObjectiveState{
		Name: "points",
		Scores: map[string]*javaScoreboardScoreState{
			"alpha": {Owner: "alpha", Value: 2},
			"beta":  {Owner: "beta", Value: 5},
		},
		Active: make(map[int32]*javaScoreboardDisplayState),
	}
	display := &javaScoreboardDisplayState{BedrockID: "objective", Slot: packet.ScoreboardSlotSidebar}
	b.mu.Lock()
	entries := b.buildScoreboardEntriesLocked(objective, display)
	b.mu.Unlock()
	if len(entries) != 2 || entries[0].DisplayName != "beta" || entries[1].DisplayName != "[P] alpha!" || entries[0].Score != 5 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].IdentityType != gtprotocol.ScoreboardIdentityFakePlayer {
		t.Fatalf("identity = %d", entries[0].IdentityType)
	}
	firstID := display.Rendered["beta"].EntryID
	objective.Scores["beta"].Value = 1
	b.mu.Lock()
	entries = b.buildScoreboardEntriesLocked(objective, display)
	b.mu.Unlock()
	if display.Rendered["beta"].EntryID != firstID {
		t.Fatalf("entry identity was not stable: before=%d after=%d", firstID, display.Rendered["beta"].EntryID)
	}
}

func TestScoreboardSlotMapping(t *testing.T) {
	for position, want := range map[int32]string{
		0: packet.ScoreboardSlotList, 1: packet.ScoreboardSlotSidebar, 2: packet.ScoreboardSlotBelowName,
		18: packet.ScoreboardSlotSidebar,
	} {
		got, ok := scoreboardSlot(position)
		if !ok || got != want {
			t.Fatalf("position %d = %q, %t; want %q", position, got, ok, want)
		}
	}
	if _, ok := scoreboardSlot(19); ok {
		t.Fatal("invalid scoreboard slot was accepted")
	}
}
