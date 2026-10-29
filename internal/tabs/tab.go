// Package tabs contains the song domain model (tab, chord, and mixed songs),
// XML codec, and file storage.
package tabs

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	// CurrentVersion is the XML schema version written for new files.
	CurrentVersion = 3
	// StandardTuning is the default tuning written to new tabs.
	StandardTuning = "E2 A2 D3 G3 B3 E4"
	// MinFret and MaxFret bound valid fret numbers.
	MinFret = 0
	MaxFret = 24
	// MinString and MaxString bound valid string numbers (1 = high E).
	MinString = 1
	MaxString = 6
	// MaxTempo is the largest accepted tempo value.
	MaxTempo = 400
	// MaxMeasures bounds the number of measures in a single tab.
	MaxMeasures = 10000
	// MaxLines bounds the number of lines in a single song.
	MaxLines = 500
	// MaxLyrics bounds the length of a line's lyrics.
	MaxLyrics = 2000
	// MaxChord bounds the length of a chord line.
	MaxChord = 200
	// MaxStrumming bounds the length of a song's strumming pattern.
	MaxStrumming = 200
	// MaxPause bounds the pause after a line in beats.
	MaxPause = 16

	// SongTypeTab is a tablature-only song.
	SongTypeTab = "tab"
	// SongTypeChord is a lyrics-and-chords-only song.
	SongTypeChord = "chord"
	// SongTypeMixed combines tablature and chord lines.
	SongTypeMixed = "mixed"

	// LineTypeTab is a six-string tablature line.
	LineTypeTab = "tab"
	// LineTypeChord is a lyrics-and-chord line.
	LineTypeChord = "chord"
)

// Beat is a single fretted note on one string. SlideTo, when set, is the
// destination fret of a slide written as "fret/slideTo".
type Beat struct {
	String  int  `xml:"string,attr" json:"string"`
	Fret    int  `xml:"fret,attr" json:"fret"`
	SlideTo *int `xml:"slideTo,attr,omitempty" json:"slideTo,omitempty"`
}

// Measure groups the notes played together in one column of tablature.
type Measure struct {
	Beats      []Beat       `xml:"beat" json:"beats"`
	ExtraAttrs []xml.Attr   `xml:",any,attr" json:"-"`
	Extra      []RawElement `xml:",any" json:"-"`
}

// Line is one song line. Tab lines hold positions on six strings; chord lines
// hold lyrics with a chord (or progression) underneath.
type Line struct {
	Type       string       `xml:"type,attr,omitempty" json:"type"`
	Lyrics     string       `xml:"lyrics,omitempty" json:"lyrics"`
	Chord      string       `xml:"chord,omitempty" json:"chord"`
	Pause      int          `xml:"pause,omitempty" json:"pause"`
	Measures   []Measure    `xml:"measure" json:"measures"`
	ExtraAttrs []xml.Attr   `xml:",any,attr" json:"-"`
	Extra      []RawElement `xml:",any" json:"-"`
}

// Tab is a complete song document.
type Tab struct {
	XMLName    xml.Name     `xml:"tab" json:"-"`
	Version    int          `xml:"version,attr" json:"version"`
	Title      string       `xml:"title,omitempty" json:"title"`
	Artist     string       `xml:"artist,omitempty" json:"artist"`
	Tuning     string       `xml:"tuning,omitempty" json:"tuning"`
	Tempo      int          `xml:"tempo,omitempty" json:"tempo"`
	Type       string       `xml:"type,omitempty" json:"type"`
	Strumming  string       `xml:"strumming,omitempty" json:"strumming"`
	Lines      []Line       `xml:"line" json:"lines"`
	ExtraAttrs []xml.Attr   `xml:",any,attr" json:"-"`
	Extra      []RawElement `xml:",any" json:"-"`

	// LegacyMeasures reads version 1 documents, which stored measures directly
	// under <tab>. Normalize moves them into a single line.
	LegacyMeasures []Measure `xml:"measure" json:"-"`
}

// RawElement preserves an unknown XML element so it survives a load/save cycle.
type RawElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	InnerXML string     `xml:",innerxml"`
}

// New returns an empty tab song with sensible defaults.
func New() Tab {
	return Tab{
		Version: CurrentVersion,
		Type:    SongTypeTab,
		Tuning:  StandardTuning,
		Tempo:   120,
		Lines:   []Line{{Type: LineTypeTab, Measures: make([]Measure, 8)}},
	}
}

// normalSongType returns the canonical song type, defaulting to tab for legacy
// documents.
func normalSongType(songType string) string {
	switch strings.ToLower(strings.TrimSpace(songType)) {
	case SongTypeChord:
		return SongTypeChord
	case SongTypeMixed:
		return SongTypeMixed
	default:
		return SongTypeTab
	}
}

// normalLineType returns the canonical line type, defaulting to tab.
func normalLineType(lineType string) string {
	if strings.EqualFold(strings.TrimSpace(lineType), LineTypeChord) {
		return LineTypeChord
	}
	return LineTypeTab
}

// Normalize fills defaults for optional fields and migrates older documents.
func (t *Tab) Normalize() {
	if t.Version != CurrentVersion {
		t.Version = CurrentVersion
	}
	if strings.TrimSpace(t.Tuning) == "" {
		t.Tuning = StandardTuning
	}
	t.Type = normalSongType(t.Type)
	t.Title = strings.TrimSpace(t.Title)
	t.Artist = strings.TrimSpace(t.Artist)
	t.Tuning = strings.TrimSpace(t.Tuning)
	t.Strumming = strings.TrimSpace(t.Strumming)

	if len(t.LegacyMeasures) > 0 {
		if len(t.Lines) == 0 {
			t.Lines = []Line{{Type: LineTypeTab, Measures: t.LegacyMeasures}}
		} else {
			t.Lines[0].Measures = append(append([]Measure{}, t.LegacyMeasures...), t.Lines[0].Measures...)
		}
		t.LegacyMeasures = nil
	}
	if len(t.Lines) == 0 {
		lineType := LineTypeTab
		var measures []Measure
		if t.Type == SongTypeChord {
			lineType = LineTypeChord
		} else {
			measures = make([]Measure, 8)
		}
		t.Lines = []Line{{Type: lineType, Measures: measures}}
	}

	for i := range t.Lines {
		line := &t.Lines[i]
		line.Chord = strings.TrimSpace(line.Chord)
		switch t.Type {
		case SongTypeTab:
			line.Type = LineTypeTab
		case SongTypeChord:
			line.Type = LineTypeChord
		default:
			if line.Type == "" {
				if line.Chord != "" && len(line.Measures) == 0 {
					line.Type = LineTypeChord
				} else {
					line.Type = LineTypeTab
				}
			} else {
				line.Type = normalLineType(line.Type)
			}
		}
		if line.Type == LineTypeChord {
			line.Measures = nil
		} else {
			line.Chord = ""
		}
	}
}

// LineType reports the effective type of the zero-based line ("tab" or
// "chord"), accounting for the song type.
func (t *Tab) LineType(index int) string {
	if index < 0 || index >= len(t.Lines) {
		return ""
	}
	line := t.Lines[index]
	switch normalSongType(t.Type) {
	case SongTypeTab:
		return LineTypeTab
	case SongTypeChord:
		return LineTypeChord
	default:
		return normalLineType(line.Type)
	}
}

// Validate checks the document against the song schema.
func (t *Tab) Validate() error {
	if t.Version != 0 && t.Version != 1 && t.Version != 2 && t.Version != CurrentVersion {
		return fmt.Errorf("unsupported song version %d", t.Version)
	}
	if utf8.RuneCountInString(t.Title) > 200 {
		return errors.New("title is too long (maximum 200 characters)")
	}
	if utf8.RuneCountInString(t.Artist) > 200 {
		return errors.New("artist is too long (maximum 200 characters)")
	}
	if utf8.RuneCountInString(t.Tuning) > 100 {
		return errors.New("tuning is too long (maximum 100 characters)")
	}
	if utf8.RuneCountInString(t.Strumming) > MaxStrumming {
		return fmt.Errorf("strumming pattern is too long (maximum %d characters)", MaxStrumming)
	}
	switch strings.ToLower(strings.TrimSpace(t.Type)) {
	case "", SongTypeTab, SongTypeChord, SongTypeMixed:
	default:
		return fmt.Errorf("unsupported song type %q", t.Type)
	}
	if t.Tempo < 0 || t.Tempo > MaxTempo {
		return fmt.Errorf("tempo must be between 0 and %d", MaxTempo)
	}
	if len(t.Lines) > MaxLines {
		return fmt.Errorf("too many lines (maximum %d)", MaxLines)
	}

	total := 0
	for li, line := range t.Lines {
		if utf8.RuneCountInString(line.Lyrics) > MaxLyrics {
			return fmt.Errorf("line %d lyrics are too long (maximum %d characters)", li+1, MaxLyrics)
		}
		if utf8.RuneCountInString(line.Chord) > MaxChord {
			return fmt.Errorf("line %d chord is too long (maximum %d characters)", li+1, MaxChord)
		}
		lineType := normalLineType(line.Type)
		switch normalSongType(t.Type) {
		case SongTypeTab:
			lineType = LineTypeTab
		case SongTypeChord:
			lineType = LineTypeChord
		default:
			if line.Type == "" {
				lineType = LineTypeTab
				if line.Chord != "" && len(line.Measures) == 0 {
					lineType = LineTypeChord
				}
			}
		}
		if lineType == LineTypeChord {
			continue
		}
		if line.Pause < 0 || line.Pause > MaxPause {
			return fmt.Errorf("line %d pause must be between 0 and %d beats", li+1, MaxPause)
		}
		total += len(line.Measures)
		if total > MaxMeasures {
			return fmt.Errorf("too many measures (maximum %d)", MaxMeasures)
		}
		for i, m := range line.Measures {
			if len(m.Beats) > MaxString {
				return fmt.Errorf("line %d measure %d has too many notes", li+1, i+1)
			}
			seen := make(map[int]bool, len(m.Beats))
			for j, b := range m.Beats {
				if b.String < MinString || b.String > MaxString {
					return fmt.Errorf("line %d measure %d beat %d: string must be between %d and %d", li+1, i+1, j+1, MinString, MaxString)
				}
				if b.Fret < MinFret || b.Fret > MaxFret {
					return fmt.Errorf("line %d measure %d beat %d: fret must be between %d and %d", li+1, i+1, j+1, MinFret, MaxFret)
				}
				if b.SlideTo != nil && (*b.SlideTo < MinFret || *b.SlideTo > MaxFret) {
					return fmt.Errorf("line %d measure %d beat %d: slide target must be between %d and %d", li+1, i+1, j+1, MinFret, MaxFret)
				}
				if seen[b.String] {
					return fmt.Errorf("line %d measure %d: string %d appears more than once", li+1, i+1, b.String)
				}
				seen[b.String] = true
			}
		}
	}
	return nil
}

// Encode writes the tab as an indented XML document.
func Encode(w io.Writer, tab *Tab) error {
	if tab == nil {
		return errors.New("song is nil")
	}
	copied := *tab
	copied.Normalize()
	if err := copied.Validate(); err != nil {
		return fmt.Errorf("invalid song: %w", err)
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(copied); err != nil {
		return fmt.Errorf("encoding song: %w", err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

// Decode reads a tab from XML.
func Decode(r io.Reader) (*Tab, error) {
	var tab Tab
	dec := xml.NewDecoder(r)
	dec.Strict = true
	if err := dec.Decode(&tab); err != nil {
		return nil, fmt.Errorf("decoding song XML: %w", err)
	}
	tab.Normalize()
	if err := tab.Validate(); err != nil {
		return nil, fmt.Errorf("invalid song: %w", err)
	}
	return &tab, nil
}
