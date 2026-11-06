package vm

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuiltInFingerRobotArmProfiles(t *testing.T) {
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		t.Fatalf("built-in profiles are invalid: %v", err)
	}

	tests := []struct {
		id        string
		name      string
		pressUS   [FingerRobotStrengthLevelCount]int
		wantLevel int
	}{
		{
			id:        FingerRobotArmProfileLongID,
			name:      "长摆臂",
			pressUS:   [FingerRobotStrengthLevelCount]int{2000, 1950, 1900, 1850, 1800},
			wantLevel: 1900,
		},
		{
			id:        FingerRobotArmProfileShortID,
			name:      "短摆臂",
			pressUS:   [FingerRobotStrengthLevelCount]int{1550, 1500, 1450, 1400, 1350},
			wantLevel: 1450,
		},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			profile, err := GetFingerRobotArmProfile(test.id)
			if err != nil {
				t.Fatalf("get profile: %v", err)
			}
			if profile.ID != test.id || profile.DisplayName != test.name {
				t.Fatalf("unexpected identity: %+v", profile)
			}
			if profile.ReleaseUS != FingerRobotSafeReleaseUS || profile.MinSafePressUS != 889 {
				t.Fatalf("unexpected safety limits: %+v", profile)
			}
			if profile.DefaultStrength != FingerRobotDefaultStrengthLevel {
				t.Fatalf("default strength = %d, want %d", profile.DefaultStrength, FingerRobotDefaultStrengthLevel)
			}
			if profile.PressUSByStrength != test.pressUS {
				t.Fatalf("press table = %v, want %v", profile.PressUSByStrength, test.pressUS)
			}

			pressUS, err := ResolveFingerRobotPressUS(test.id, profile.DefaultStrength)
			if err != nil {
				t.Fatalf("resolve default strength: %v", err)
			}
			if pressUS != test.wantLevel {
				t.Fatalf("default press pulse = %d, want %d", pressUS, test.wantLevel)
			}
		})
	}
}

func TestFingerRobotArmProfileWhitelistAndStrengthBounds(t *testing.T) {
	for _, id := range []string{"", "medium", "LONG", "short "} {
		if _, err := GetFingerRobotArmProfile(id); !errors.Is(err, ErrFingerRobotInvalid) {
			t.Errorf("GetFingerRobotArmProfile(%q) error = %v, want ErrFingerRobotInvalid", id, err)
		}
		if _, err := ResolveFingerRobotPressUS(id, FingerRobotDefaultStrengthLevel); !errors.Is(err, ErrFingerRobotInvalid) {
			t.Errorf("ResolveFingerRobotPressUS(%q) error = %v, want ErrFingerRobotInvalid", id, err)
		}
	}

	for _, strength := range []int{-1, 0, FingerRobotStrengthLevelCount + 1, 100} {
		if _, err := ResolveFingerRobotPressUS(FingerRobotArmProfileShortID, strength); !errors.Is(err, ErrFingerRobotInvalid) {
			t.Errorf("strength %d error = %v, want ErrFingerRobotInvalid", strength, err)
		}
	}

	for strength, want := range []int{1550, 1500, 1450, 1400, 1350} {
		got, err := ResolveFingerRobotPressUS(FingerRobotArmProfileShortID, strength+1)
		if err != nil {
			t.Fatalf("resolve strength %d: %v", strength+1, err)
		}
		if got != want {
			t.Errorf("strength %d = %d, want %d", strength+1, got, want)
		}
	}
}

func TestFingerRobotArmProfileReturnsAreCopies(t *testing.T) {
	profile, err := GetFingerRobotArmProfile(FingerRobotArmProfileLongID)
	if err != nil {
		t.Fatal(err)
	}
	profile.DisplayName = "changed"
	profile.PressUSByStrength[0] = 889

	again, err := GetFingerRobotArmProfile(FingerRobotArmProfileLongID)
	if err != nil {
		t.Fatal(err)
	}
	if again.DisplayName != "长摆臂" || again.PressUSByStrength[0] != 2000 {
		t.Fatalf("getter exposed registry mutation: %+v", again)
	}

	first, err := ListFingerRobotArmProfiles()
	if err != nil {
		t.Fatal(err)
	}
	second, err := ListFingerRobotArmProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("profile list lengths = %d/%d, want 2/2", len(first), len(second))
	}
	if got := []string{first[0].ID, first[1].ID}; !reflect.DeepEqual(got, []string{"long", "short"}) {
		t.Fatalf("list order = %v, want [long short]", got)
	}
	first[0].ID = "changed"
	first[1].PressUSByStrength[4] = 889
	if second[0].ID != FingerRobotArmProfileLongID || second[1].PressUSByStrength[4] != 1350 {
		t.Fatalf("list calls share mutable data: %+v", second)
	}
	third, err := ListFingerRobotArmProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if third[0].ID != FingerRobotArmProfileLongID || third[1].PressUSByStrength[4] != 1350 {
		t.Fatalf("list exposed registry mutation: %+v", third)
	}
}

func TestValidateFingerRobotArmProfilesRejectsInvalidTables(t *testing.T) {
	valid := cloneBuiltInFingerRobotArmProfiles()
	tests := []struct {
		name   string
		mutate func([]FingerRobotArmProfile) []FingerRobotArmProfile
	}{
		{
			name: "empty ID",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].ID = ""
				return profiles
			},
		},
		{
			name: "unknown ID",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].ID = "medium"
				return profiles
			},
		},
		{
			name: "missing profile",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				return profiles[:1]
			},
		},
		{
			name: "duplicate ID",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[1] = profiles[0]
				return profiles
			},
		},
		{
			name: "empty display name",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].DisplayName = ""
				return profiles
			},
		},
		{
			name: "wrong display name",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].DisplayName = "Long arm"
				return profiles
			},
		},
		{
			name: "wrong release pulse",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].ReleaseUS--
				return profiles
			},
		},
		{
			name: "wrong minimum safe pulse",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].MinSafePressUS++
				return profiles
			},
		},
		{
			name: "default strength below range",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].DefaultStrength = 0
				return profiles
			},
		},
		{
			name: "default strength above range",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].DefaultStrength = FingerRobotStrengthLevelCount + 1
				return profiles
			},
		},
		{
			name: "non-L3 default strength",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].DefaultStrength = 2
				return profiles
			},
		},
		{
			name: "press pulse below minimum",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[1].PressUSByStrength[4] = FingerRobotArmProfileMinSafePressUS - 1
				return profiles
			},
		},
		{
			name: "press pulse above maximum",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].PressUSByStrength[0] = fingerRobotArmProfileMaxPressUS + 1
				return profiles
			},
		},
		{
			name: "duplicate press pulses",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[0].PressUSByStrength[2] = profiles[0].PressUSByStrength[1]
				return profiles
			},
		},
		{
			name: "increasing press pulses",
			mutate: func(profiles []FingerRobotArmProfile) []FingerRobotArmProfile {
				profiles[1].PressUSByStrength[3] = profiles[1].PressUSByStrength[2] + 1
				return profiles
			},
		},
	}

	if err := validateFingerRobotArmProfiles(valid); err != nil {
		t.Fatalf("valid fixture failed validation: %v", err)
	}
	boundary := cloneBuiltInFingerRobotArmProfiles()
	boundary[0].PressUSByStrength = [FingerRobotStrengthLevelCount]int{2000, 1999, 1500, 890, 889}
	if err := validateFingerRobotArmProfiles(boundary); err != nil {
		t.Fatalf("inclusive safety boundaries failed validation: %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := test.mutate(cloneBuiltInFingerRobotArmProfiles())
			if err := validateFingerRobotArmProfiles(profiles); !errors.Is(err, ErrFingerRobotInvalid) {
				t.Fatalf("validation error = %v, want ErrFingerRobotInvalid", err)
			}
			registry, err := initializeFingerRobotArmProfiles(profiles)
			if !errors.Is(err, ErrFingerRobotInvalid) || registry != nil {
				t.Fatalf("initialization = (%v, %v), want (nil, ErrFingerRobotInvalid)", registry, err)
			}
		})
	}
}

func cloneBuiltInFingerRobotArmProfiles() []FingerRobotArmProfile {
	return append([]FingerRobotArmProfile(nil), builtInFingerRobotArmProfiles[:]...)
}
