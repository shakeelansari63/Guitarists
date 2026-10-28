package tabs

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func intPtr(value int) *int { return &value }

func sampleTab() Tab {
	return Tab{
		Version: CurrentVersion,
		Title:   "Riff",
		Artist:  "Player",
		Tuning:  StandardTuning,
		Tempo:   140,
		Lines: []Line{
			{
				Lyrics: "first line of the song",
				Pause:  1,
				Measures: []Measure{
					{Beats: []Beat{{String: 6, Fret: 0}, {String: 5, Fret: 3}}},
					{Beats: []Beat{{String: 4, Fret: 12}}},
					{},
				},
			},
			{
				Lyrics: "second line",
				Measures: []Measure{
					{Beats: []Beat{{String: 1, Fret: 7, SlideTo: intPtr(9)}}},
				},
			},
		},
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()

	in := sampleTab()
	var buf bytes.Buffer
	if err := Encode(&buf, &in); err != nil {
		t.Fatalf("encode: %v", err)
	}

	out, err := Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Title != in.Title || out.Artist != in.Artist || out.Tempo != in.Tempo {
		t.Fatalf("metadata mismatch: %+v", out)
	}
	if len(out.Lines) != len(in.Lines) || len(out.Lines[0].Measures) != len(in.Lines[0].Measures) {
		t.Fatalf("line mismatch: %+v", out.Lines)
	}
	if out.Lines[0].Lyrics != in.Lines[0].Lyrics || out.Lines[1].Lyrics != in.Lines[1].Lyrics {
		t.Fatalf("lyrics mismatch: %+v", out.Lines)
	}
	if out.Lines[0].Pause != 1 {
		t.Fatalf("pause = %d, want 1", out.Lines[0].Pause)
	}
	slide := out.Lines[1].Measures[0].Beats[0].SlideTo
	if slide == nil || *slide != 9 {
		t.Fatalf("slide target = %v, want 9", slide)
	}
	if got := out.Lines[0].Measures[0].Beats[1]; got.String != 5 || got.Fret != 3 {
		t.Fatalf("beat mismatch: %+v", got)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	t.Parallel()

	var tab Tab
	tab.Normalize()
	if tab.Version != CurrentVersion || tab.Tuning != StandardTuning {
		t.Fatalf("unexpected defaults: %+v", tab)
	}
	if tab.Type != SongTypeTab || tab.Lines[0].Type != LineTypeTab {
		t.Fatalf("unexpected song type defaults: %+v", tab)
	}
	if len(tab.Lines) != 1 || len(tab.Lines[0].Measures) != 8 {
		t.Fatalf("unexpected default lines: %+v", tab.Lines)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tab  Tab
		ok   bool
	}{
		{name: "valid", tab: sampleTab(), ok: true},
		{name: "empty lines", tab: Tab{Version: CurrentVersion}, ok: true},
		{name: "legacy version", tab: Tab{Version: 1}, ok: true},
		{name: "bad version", tab: Tab{Version: 4}, ok: false},
		{name: "bad song type", tab: Tab{Version: CurrentVersion, Type: "riff"}, ok: false},
		{name: "long strumming", tab: Tab{Version: CurrentVersion, Strumming: strings.Repeat("D", MaxStrumming+1)}, ok: false},
		{name: "long chord", tab: Tab{Version: CurrentVersion, Type: SongTypeChord, Lines: []Line{{Type: LineTypeChord, Chord: strings.Repeat("G", MaxChord+1)}}}, ok: false},
		{name: "bad string low", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 0, Fret: 0}}}}}}}, ok: false},
		{name: "bad string high", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 7, Fret: 0}}}}}}}, ok: false},
		{name: "bad fret low", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: -1}}}}}}}, ok: false},
		{name: "bad fret high", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 25}}}}}}}, ok: false},
		{name: "duplicate string", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 0}, {String: 1, Fret: 2}}}}}}}, ok: false},
		{name: "bad slide low", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 5, SlideTo: intPtr(-1)}}}}}}}, ok: false},
		{name: "bad slide high", tab: Tab{Version: CurrentVersion, Lines: []Line{{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 5, SlideTo: intPtr(25)}}}}}}}, ok: false},
		{name: "bad pause", tab: Tab{Version: CurrentVersion, Lines: []Line{{Pause: MaxPause + 1}}}, ok: false},
		{name: "bad tempo", tab: Tab{Version: CurrentVersion, Tempo: MaxTempo + 1}, ok: false},
		{name: "long title", tab: Tab{Version: CurrentVersion, Title: strings.Repeat("a", 201)}, ok: false},
		{name: "long lyrics", tab: Tab{Version: CurrentVersion, Lines: []Line{{Lyrics: strings.Repeat("a", MaxLyrics+1)}}}, ok: false},
		{name: "too many lines", tab: Tab{Version: CurrentVersion, Lines: make([]Line, MaxLines+1)}, ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.tab.Validate()
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestUnknownElementsPreserved(t *testing.T) {
	t.Parallel()

	input := `<?xml version="1.0" encoding="UTF-8"?>
<tab version="2" custom="keep">
  <title>Song</title>
  <tuning>E2 A2 D3 G3 B3 E4</tuning>
  <tempo>100</tempo>
  <vendor><name>Example</name></vendor>
  <line note="keep">
    <lyrics>Hello</lyrics>
    <measure><beat string="1" fret="5"/><riff x="1">raw</riff></measure>
  </line>
</tab>`

	tab, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var buf bytes.Buffer
	if err := Encode(&buf, tab); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	for _, want := range []string{`custom="keep"`, `note="keep"`, "<vendor><name>Example</name></vendor>", `<riff x="1">raw</riff>`, "<lyrics>Hello</lyrics>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}

	var check struct {
		Custom string `xml:"custom,attr"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if check.Custom != "keep" {
		t.Fatalf("custom attribute = %q", check.Custom)
	}
}

func TestDecodeVersion1MigratesToLine(t *testing.T) {
	t.Parallel()

	input := `<?xml version="1.0" encoding="UTF-8"?>
<tab version="1">
  <title>Old song</title>
  <measure><beat string="6" fret="3"/></measure>
  <measure><beat string="5" fret="0"/></measure>
</tab>`

	tab, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tab.Version != CurrentVersion {
		t.Fatalf("version = %d, want %d", tab.Version, CurrentVersion)
	}
	if len(tab.Lines) != 1 || len(tab.Lines[0].Measures) != 2 {
		t.Fatalf("unexpected lines: %+v", tab.Lines)
	}
	if got := tab.Lines[0].Measures[0].Beats[0]; got.String != 6 || got.Fret != 3 {
		t.Fatalf("beat mismatch: %+v", got)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, tab); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(buf.String(), "<tab version=\"1\"") || !strings.Contains(buf.String(), "<line") {
		t.Fatalf("expected migrated version 3 output:\n%s", buf.String())
	}
	if got := strings.Count(buf.String(), "<measure>"); got != 2 {
		t.Fatalf("got %d migrated measures, want 2:\n%s", got, buf.String())
	}
}

func TestChordAndMixedRoundTrip(t *testing.T) {
	t.Parallel()

	song := Tab{
		Version:   CurrentVersion,
		Title:     "Chord song",
		Artist:    "Player",
		Tuning:    StandardTuning,
		Tempo:     90,
		Type:      SongTypeMixed,
		Strumming: "D DU UDU",
		Lines: []Line{
			{Type: LineTypeChord, Lyrics: "first line", Chord: "G A Em"},
			{Type: LineTypeTab, Lyrics: "riff", Measures: []Measure{{Beats: []Beat{{String: 6, Fret: 3}}}}},
			{Type: LineTypeChord, Chord: "Bsus2"},
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, &song); err != nil {
		t.Fatalf("encode: %v", err)
	}
	encoded := buf.String()
	out, err := Decode(strings.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Type != SongTypeMixed || out.Strumming != "D DU UDU" {
		t.Fatalf("song metadata mismatch: %+v", out)
	}
	if len(out.Lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(out.Lines))
	}
	if out.LineType(0) != LineTypeChord || out.Lines[0].Chord != "G A Em" {
		t.Fatalf("first line mismatch: %+v", out.Lines[0])
	}
	if out.LineType(1) != LineTypeTab || len(out.Lines[1].Measures) != 1 {
		t.Fatalf("second line mismatch: %+v", out.Lines[1])
	}
	if out.LineType(2) != LineTypeChord || out.Lines[2].Chord != "Bsus2" {
		t.Fatalf("third line mismatch: %+v", out.Lines[2])
	}
	for _, want := range []string{`<tab version="3"`, `<type>mixed</type>`, `<strumming>D DU UDU</strumming>`, `<line type="chord">`, `<line type="tab">`} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("output missing %q:\n%s", want, encoded)
		}
	}
}

func TestChordOnlyNormalizesLines(t *testing.T) {
	t.Parallel()

	song := Tab{
		Version: CurrentVersion,
		Type:    SongTypeChord,
		Lines:   []Line{{Lyrics: "words", Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 0}}}}}},
	}
	song.Normalize()
	if song.Type != SongTypeChord || song.LineType(0) != LineTypeChord {
		t.Fatalf("unexpected normalized song: %+v", song)
	}
	if len(song.Lines[0].Measures) != 0 {
		t.Fatalf("chord line should not keep measures: %+v", song.Lines[0])
	}
	if err := song.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestDecodeLegacyTabDefaultsToTabType(t *testing.T) {
	t.Parallel()

	input := `<?xml version="1.0" encoding="UTF-8"?>
<tab version="2">
  <title>Old song</title>
  <line><lyrics>words</lyrics><measure><beat string="1" fret="0"/></measure></line>
</tab>`
	song, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if song.Type != SongTypeTab || song.LineType(0) != LineTypeTab {
		t.Fatalf("legacy song should default to tab: %+v", song)
	}
}

func TestStoreSummaryType(t *testing.T) {
	t.Parallel()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	song := Tab{Version: CurrentVersion, Title: "Chords", Type: SongTypeChord, Lines: []Line{{Type: LineTypeChord, Chord: "G"}}}
	if _, err := store.Save("Chords", &song); err != nil {
		t.Fatalf("save: %v", err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Type != SongTypeChord {
		t.Fatalf("unexpected summary: %+v", list)
	}
}

func TestStoreSaveLoadListDelete(t *testing.T) {
	t.Parallel()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	tab := sampleTab()
	summary, err := store.Save("My Riff", &tab)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if summary.Name != "My Riff.xml" {
		t.Fatalf("summary name = %q", summary.Name)
	}

	loaded, err := store.Load("My Riff")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Title != tab.Title {
		t.Fatalf("loaded title = %q", loaded.Title)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "My Riff.xml" || list[0].Title != "Riff" {
		t.Fatalf("unexpected list: %+v", list)
	}

	if err := store.Delete("My Riff.xml"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir(), "My Riff.xml")); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err = %v", err)
	}
}

func TestStoreLoadMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.xml"), []byte("<tab><measure>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("broken.xml"); err == nil {
		t.Fatal("expected malformed XML to fail")
	}
}

func TestStoreRejectsTraversal(t *testing.T) {
	t.Parallel()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	for _, name := range []string{"../evil", "nested/file", `nested\file`, "", " ", ".hidden", "bad:name", "trailing."} {
		if _, err := store.Save(name, &Tab{Version: 1}); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "Song", want: "Song.xml"},
		{in: "Song.xml", want: "Song.xml"},
		{in: "Song.XML", want: "Song.XML"},
		{in: "  Spacey Name  ", want: "Spacey Name.xml"},
		{in: "AC/DC", wantErr: true},
		{in: ".hidden", wantErr: true},
		{in: "", wantErr: true},
		{in: strings.Repeat("a", maxNameLen), wantErr: true},
	}
	for _, tt := range tests {
		got, err := NormalizeName(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeName(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Fatalf("NormalizeName(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}
