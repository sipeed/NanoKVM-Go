// Package keyboardlayout translates text to USB keyboard strokes for a target layout.
package keyboardlayout

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

const Default = "en-US"
const microsoftPinyin = "zh-CN"

type Stroke struct {
	Modifiers byte
	Code      byte
}

func (s Stroke) Report() []byte {
	return []byte{s.Modifiers, 0, s.Code, 0, 0, 0, 0, 0}
}

type keyRun struct {
	modifiers  byte
	firstCode  byte
	characters string
}

type accentKey struct {
	stroke  Stroke
	bases   string
	outputs string
}

type profile struct {
	direct  []keyRun
	accents []accentKey
}

type Layout struct {
	ID   string
	keys map[rune][]Stroke
}

var layouts = make(map[string]Layout)
var usLabels = make(map[Stroke]string)

func init() {
	for id, profile := range profiles {
		keys := map[rune][]Stroke{'\n': {{0, 40}}, '\t': {{0, 43}}}
		for _, run := range profile.direct {
			for offset, char := range []rune(run.characters) {
				keys[char] = []Stroke{{run.modifiers, run.firstCode + byte(offset)}}
			}
		}
		for _, accent := range profile.accents {
			outputs := []rune(accent.outputs)
			for i, base := range []rune(accent.bases) {
				if keys[outputs[i]] == nil && len(keys[base]) == 1 {
					keys[outputs[i]] = []Stroke{accent.stroke, keys[base][0]}
				}
			}
		}
		layouts[id] = Layout{ID: id, keys: keys}
	}
	// Full Pinyin in Chinese mode accepts vuc + a Unicode code point + Space.
	layouts[microsoftPinyin] = Layout{ID: microsoftPinyin, keys: layouts[Default].keys}
	// The HID activity display labels physical key positions using the US legends.
	for char, strokes := range layouts[Default].keys {
		usLabels[strokes[0]] = string(char)
	}
}

func USLabel(s Stroke) string { return usLabels[s] }

func Lookup(id string) (Layout, bool) {
	layout, ok := layouts[id]
	return layout, ok
}

func Load(path string) (Layout, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return layouts[Default], nil
	}
	if err != nil {
		return Layout{}, err
	}
	layout, ok := Lookup(strings.TrimSpace(string(data)))
	if !ok {
		return Layout{}, fmt.Errorf("unknown saved keyboard layout")
	}
	return layout, nil
}

// Encode validates the entire string before the caller sends any keyboard reports.
func (l Layout) Encode(text string) ([]Stroke, error) {
	var strokes []Stroke
	for _, char := range strings.ReplaceAll(text, "\r\n", "\n") {
		if l.ID == microsoftPinyin && char != '\n' && char != '\t' {
			// Encode punctuation and Latin letters too: literal US key presses can
			// start a composition or turn into full-width punctuation in Chinese mode.
			if char > 0xffff || !unicode.IsPrint(char) {
				return nil, fmt.Errorf("Microsoft Pinyin V-mode cannot type %U with this profile", char)
			}
			for _, key := range fmt.Sprintf("vuc%04x ", char) {
				strokes = append(strokes, l.keys[key][0])
			}
			continue
		}
		keys := l.keys[char]
		if len(keys) == 0 {
			return nil, fmt.Errorf("keyboard layout %s cannot type %U", l.ID, char)
		}
		strokes = append(strokes, keys...)
	}
	return strokes, nil
}
