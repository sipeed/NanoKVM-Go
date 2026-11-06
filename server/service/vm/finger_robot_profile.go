package vm

import "fmt"

const (
	FingerRobotArmProfileLongID  = "long"
	FingerRobotArmProfileShortID = "short"

	FingerRobotStrengthLevelCount   = 5
	FingerRobotDefaultStrengthLevel = 3

	FingerRobotArmProfileReleaseUS      = FingerRobotSafeReleaseUS
	FingerRobotArmProfileMinSafePressUS = 889
	fingerRobotArmProfileMaxPressUS     = 2000
)

// FingerRobotArmProfile is a fixed, product-defined mapping from a user-facing
// strength level to a safe PWM pulse. PressUSByStrength uses zero-based array
// indexes internally for the user-facing levels 1 through 5.
type FingerRobotArmProfile struct {
	ID                string
	DisplayName       string
	ReleaseUS         int
	MinSafePressUS    int
	PressUSByStrength [FingerRobotStrengthLevelCount]int
	DefaultStrength   int
}

var builtInFingerRobotArmProfiles = [...]FingerRobotArmProfile{
	{
		ID:                FingerRobotArmProfileLongID,
		DisplayName:       "长摆臂",
		ReleaseUS:         FingerRobotArmProfileReleaseUS,
		MinSafePressUS:    FingerRobotArmProfileMinSafePressUS,
		PressUSByStrength: [FingerRobotStrengthLevelCount]int{2000, 1950, 1900, 1850, 1800},
		DefaultStrength:   FingerRobotDefaultStrengthLevel,
	},
	{
		ID:                FingerRobotArmProfileShortID,
		DisplayName:       "短摆臂",
		ReleaseUS:         FingerRobotArmProfileReleaseUS,
		MinSafePressUS:    FingerRobotArmProfileMinSafePressUS,
		PressUSByStrength: [FingerRobotStrengthLevelCount]int{1550, 1500, 1450, 1400, 1350},
		DefaultStrength:   FingerRobotDefaultStrengthLevel,
	},
}

var (
	fingerRobotArmProfileRegistry,
	fingerRobotArmProfilesValidationErr = initializeFingerRobotArmProfiles(builtInFingerRobotArmProfiles[:])
)

// ValidateFingerRobotArmProfiles reports whether the built-in product table
// passed package-initialization validation. Startup code must treat an error as
// fail-closed and must not initialize or move the finger robot.
func ValidateFingerRobotArmProfiles() error {
	return fingerRobotArmProfilesValidationErr
}

// GetFingerRobotArmProfile returns a value copy from the built-in whitelist.
// In particular, callers cannot mutate the registry through the five-level
// array in the returned value.
func GetFingerRobotArmProfile(id string) (FingerRobotArmProfile, error) {
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		return FingerRobotArmProfile{}, err
	}

	profile, ok := fingerRobotArmProfileRegistry[id]
	if !ok {
		return FingerRobotArmProfile{}, fmt.Errorf("%w: unknown arm profile %q", ErrFingerRobotInvalid, id)
	}
	return profile, nil
}

// ListFingerRobotArmProfiles returns value copies in the stable long, short
// product order. A new slice is allocated for every call.
func ListFingerRobotArmProfiles() ([]FingerRobotArmProfile, error) {
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		return nil, err
	}

	profiles := make([]FingerRobotArmProfile, 0, len(builtInFingerRobotArmProfiles))
	for _, id := range [...]string{FingerRobotArmProfileLongID, FingerRobotArmProfileShortID} {
		profiles = append(profiles, fingerRobotArmProfileRegistry[id])
	}
	return profiles, nil
}

// ResolveFingerRobotPressUS safely resolves a one-based strength level through
// the built-in profile whitelist. It deliberately does not accept an arbitrary
// FingerRobotArmProfile value, so callers cannot bypass startup validation.
func ResolveFingerRobotPressUS(profileID string, strength int) (int, error) {
	profile, err := GetFingerRobotArmProfile(profileID)
	if err != nil {
		return 0, err
	}
	if strength < 1 || strength > FingerRobotStrengthLevelCount {
		return 0, fmt.Errorf(
			"%w: strength must be between 1 and %d",
			ErrFingerRobotInvalid,
			FingerRobotStrengthLevelCount,
		)
	}
	return profile.PressUSByStrength[strength-1], nil
}

func initializeFingerRobotArmProfiles(profiles []FingerRobotArmProfile) (map[string]FingerRobotArmProfile, error) {
	if err := validateFingerRobotArmProfiles(profiles); err != nil {
		return nil, err
	}

	registry := make(map[string]FingerRobotArmProfile, len(profiles))
	for _, profile := range profiles {
		registry[profile.ID] = profile
	}
	return registry, nil
}

func validateFingerRobotArmProfiles(profiles []FingerRobotArmProfile) error {
	seen := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		if profile.ID == "" {
			return fmt.Errorf("%w: arm profile ID must not be empty", ErrFingerRobotInvalid)
		}
		if profile.DisplayName == "" {
			return fmt.Errorf("%w: arm profile %q display name must not be empty", ErrFingerRobotInvalid, profile.ID)
		}
		if _, ok := seen[profile.ID]; ok {
			return fmt.Errorf("%w: duplicate arm profile %q", ErrFingerRobotInvalid, profile.ID)
		}
		seen[profile.ID] = struct{}{}

		expectedDisplayName, ok := fingerRobotArmProfileDisplayName(profile.ID)
		if !ok {
			return fmt.Errorf("%w: unknown arm profile %q", ErrFingerRobotInvalid, profile.ID)
		}
		if profile.DisplayName != expectedDisplayName {
			return fmt.Errorf(
				"%w: arm profile %q display name must be %q",
				ErrFingerRobotInvalid,
				profile.ID,
				expectedDisplayName,
			)
		}
		if profile.ReleaseUS != FingerRobotArmProfileReleaseUS {
			return fmt.Errorf(
				"%w: arm profile %q release pulse must be %d us",
				ErrFingerRobotInvalid,
				profile.ID,
				FingerRobotArmProfileReleaseUS,
			)
		}
		if profile.MinSafePressUS != FingerRobotArmProfileMinSafePressUS {
			return fmt.Errorf(
				"%w: arm profile %q minimum safe press pulse must be %d us",
				ErrFingerRobotInvalid,
				profile.ID,
				FingerRobotArmProfileMinSafePressUS,
			)
		}
		if profile.DefaultStrength < 1 || profile.DefaultStrength > FingerRobotStrengthLevelCount {
			return fmt.Errorf(
				"%w: arm profile %q default strength must be between 1 and %d",
				ErrFingerRobotInvalid,
				profile.ID,
				FingerRobotStrengthLevelCount,
			)
		}
		if profile.DefaultStrength != FingerRobotDefaultStrengthLevel {
			return fmt.Errorf(
				"%w: arm profile %q default strength must be %d",
				ErrFingerRobotInvalid,
				profile.ID,
				FingerRobotDefaultStrengthLevel,
			)
		}

		previousPressUS := profile.ReleaseUS
		for index, pressUS := range profile.PressUSByStrength {
			strength := index + 1
			if pressUS < profile.MinSafePressUS || pressUS > fingerRobotArmProfileMaxPressUS {
				return fmt.Errorf(
					"%w: arm profile %q strength %d pulse must be between %d and %d us",
					ErrFingerRobotInvalid,
					profile.ID,
					strength,
					profile.MinSafePressUS,
					fingerRobotArmProfileMaxPressUS,
				)
			}
			if pressUS >= previousPressUS {
				return fmt.Errorf(
					"%w: arm profile %q strength pulses must be strictly decreasing at level %d",
					ErrFingerRobotInvalid,
					profile.ID,
					strength,
				)
			}
			previousPressUS = pressUS
		}
	}

	for _, id := range [...]string{FingerRobotArmProfileLongID, FingerRobotArmProfileShortID} {
		if _, ok := seen[id]; !ok {
			return fmt.Errorf("%w: missing arm profile %q", ErrFingerRobotInvalid, id)
		}
	}
	return nil
}

func fingerRobotArmProfileDisplayName(id string) (string, bool) {
	switch id {
	case FingerRobotArmProfileLongID:
		return "长摆臂", true
	case FingerRobotArmProfileShortID:
		return "短摆臂", true
	default:
		return "", false
	}
}
