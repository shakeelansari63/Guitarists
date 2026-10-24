// Package tabs contains the tablature domain model, XML codec, and file storage.
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
	CurrentVersion = 2
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
	// MaxLines bounds the number of tablature lines in a single tab.
	MaxLines = 500
	// MaxLyrics bounds the length of a line's lyrics.
	MaxLyrics = 2000
	// MaxPause bounds the pause after a line in beats.
	MaxPause = 16
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

// Line is one tablature staff: lyrics followed by the positions on six strings.
type Line struct {
	Lyrics     string       `xml:"lyrics,omitempty" json:"lyrics"`
	Pause      int          `xml:"pause,omitempty" json:"pause"`
	Measures   []Measure    `xml:"measure" json:"measures"`
	ExtraAttrs []xml.Attr   `xml:",any,attr" json:"-"`
	Extra      []RawElement `xml:",any" json:"-"`
}

// Tab is a complete tablature document.
type Tab struct {
	XMLName    xml.Name     `xml:"tab" json:"-"`
	Version    int          `xml:"version,attr" json:"version"`
	Title      string       `xml:"title,omitempty" json:"title"`
	Artist     string       `xml:"artist,omitempty" json:"artist"`
	Tuning     string       `xml:"tuning,omitempty" json:"tuning"`
	Tempo      int          `xml:"tempo,omitempty" json:"tempo"`
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

// New returns an empty tab with sensible defaults.
func New() Tab {
	return Tab{
		Version: CurrentVersion,
		Tuning:  StandardTuning,
		Tempo:   120,
		Lines:   []Line{{Measures: make([]Measure, 8)}},
	}
}

// Normalize fills defaults for optional fields.
func (t *Tab) Normalize() {
	if t.Version == 0 || t.Version == 1 {
		t.Version = CurrentVersion
	}
	if strings.TrimSpace(t.Tuning) == "" {
		t.Tuning = StandardTuning
	}
	t.Title = strings.TrimSpace(t.Title)
	t.Artist = strings.TrimSpace(t.Artist)
	t.Tuning = strings.TrimSpace(t.Tuning)

	if len(t.LegacyMeasures) > 0 {
		if len(t.Lines) == 0 {
			t.Lines = []Line{{Measures: t.LegacyMeasures}}
		} else {
			t.Lines[0].Measures = append(append([]Measure{}, t.LegacyMeasures...), t.Lines[0].Measures...)
		}
		t.LegacyMeasures = nil
	}
	if len(t.Lines) == 0 {
		t.Lines = []Line{{Measures: make([]Measure, 8)}}
	}
}

// Validate checks the document against the tablature schema.
func (t *Tab) Validate() error {
	if t.Version != 0 && t.Version != 1 && t.Version != CurrentVersion {
		return fmt.Errorf("unsupported tab version %d", t.Version)
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
		return errors.New("tab is nil")
	}
	copied := *tab
	copied.Normalize()
	if err := copied.Validate(); err != nil {
		return fmt.Errorf("invalid tab: %w", err)
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(copied); err != nil {
		return fmt.Errorf("encoding tab: %w", err)
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
		return nil, fmt.Errorf("decoding tab XML: %w", err)
	}
	tab.Normalize()
	if err := tab.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tab: %w", err)
	}
	return &tab, nil
}
