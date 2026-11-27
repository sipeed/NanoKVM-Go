package keyboardlayout

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCanadianPunctuation(t *testing.T) {
	tests := []struct {
		id   string
		want []Stroke
	}{
		{"en-US", []Stroke{{2, 32}, {2, 49}, {0, 56}, {0, 49}, {2, 31}}},
		{"fr-CA", []Stroke{{0, 53}, {2, 53}, {2, 32}, {64, 53}, {64, 31}}},
		{"fr-CA-multilingual", []Stroke{{2, 32}, {64, 53}, {0, 53}, {2, 53}, {2, 31}}},
	}
	for _, test := range tests {
		got, err := layouts[test.id].Encode("#|/\\@")
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: %v, %v", test.id, got, err)
		}
	}
}

func TestDeadKeysAndAccents(t *testing.T) {
	got, err := layouts["fr-CA"].Encode("^éà")
	want := []Stroke{{0, 47}, {0, 44}, {0, 56}, {0, 52}, {0, 4}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestCommandCharacters(t *testing.T) {
	for id, layout := range layouts {
		// The Latin and Pinyin profiles support all printable ASCII.
		_, hasLatin := layout.keys['a']
		for char := rune(32); hasLatin && char <= 126; char++ {
			if _, err := layout.Encode(string(char)); err != nil {
				t.Errorf("%s: %v", id, err)
			}
		}
		strokes, err := layout.Encode("before🙂after")
		if err == nil || len(strokes) != 0 {
			t.Errorf("%s accepted unsupported text", id)
		}
		got, err := layout.Encode("\r\n\t")
		if err != nil || !reflect.DeepEqual(got, []Stroke{{0, 40}, {0, 43}}) {
			t.Errorf("%s: %v, %v", id, got, err)
		}
	}
}

func TestLoadLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyboard-layout")
	layout, err := Load(path)
	if err != nil || layout.ID != Default {
		t.Fatalf("default: %v, %v", layout.ID, err)
	}
	if err := os.WriteFile(path, []byte("fr-CA\n"), 0600); err != nil {
		t.Fatal(err)
	}
	layout, err = Load(path)
	if err != nil || layout.ID != "fr-CA" {
		t.Fatalf("saved: %v, %v", layout.ID, err)
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("accepted invalid layout")
	}
}

func TestNonLatinProfiles(t *testing.T) {
	for _, tc := range []struct {
		id, text string
		want     []Stroke
	}{
		{"ru-RU", "Яё", []Stroke{{2, 29}, {0, 53}}},
		{"fa-IR", "سلام", []Stroke{{0, 22}, {0, 10}, {0, 11}, {0, 15}}},
		{"fa-IR-standard", "می‌", []Stroke{{0, 15}, {0, 7}, {2, 5}}},
	} {
		got, err := layouts[tc.id].Encode(tc.text)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, %v", tc.id, got, err)
		}
		if got, err := layouts[tc.id].Encode(tc.text + "a"); err == nil || len(got) != 0 {
			t.Errorf("%s accepted untypable Latin text", tc.id)
		}
	}
	got, err := layouts[microsoftPinyin].Encode("玉中#")
	want, _ := layouts[Default].Encode("vuc7389 vuc4e2d vuc0023 ")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Pinyin Unicode sequence: %v, %v", got, err)
	}
}
