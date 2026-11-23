package ai

import (
	"math"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/vm"
)

const (
	autoInputProfileID    = "automatic-crop-calibration"
	defaultInputProfileID = "default-no-calibration"
	absoluteMouseMode     = "absolute"

	homographyCorners     = 4
	homographySize        = 8
	homographyPivotCutoff = 1e-10

	calibrationCornerTopLeft     = "top-left"
	calibrationCornerTopRight    = "top-right"
	calibrationCornerBottomRight = "bottom-right"
	calibrationCornerBottomLeft  = "bottom-left"
)

// loadInputProfileConfig is a variable so tests can replace the profile store.
var loadInputProfileConfig = vm.LoadInputProfileConfig

type normalizedPoint struct {
	x float64
	y float64
}

// calibratedAbsolutePoint mirrors the browser input pipeline in
// web/src/pages/desktop/mouse/absolute.ts: a content-area point becomes a source
// frame point and the selected input profile calibration is applied on top.
//
// MCP has to send the same HID value the web page sends for the same visible
// point. The built-in automatic profile maps the detected content rectangle onto
// the full HID range, so on such a device the content coordinate is the HID
// coordinate.
func calibratedAbsolutePoint(area contentArea, x float64, y float64) (float64, float64) {
	return absolutePointMapper(area)(x, y)
}

func absolutePointMapper(area contentArea) func(float64, float64) (float64, float64) {
	config, _, err := loadInputProfileConfig()
	if err != nil || config == nil {
		return area.toSource
	}

	homography, ok := resolveProfileHomography(config, area)
	if !ok {
		return area.toSource
	}

	return func(x, y float64) (float64, float64) {
		sourceX, sourceY := area.toSource(x, y)
		return applyHomography(homography, normalizedPoint{x: sourceX, y: sourceY})
	}
}

// resolveProfileHomography returns the projective calibration for absolute
// mouse input, mirroring web/src/lib/input-profile.ts.
func resolveProfileHomography(config *proto.InputProfileConfig, area contentArea) ([homographySize]float64, bool) {
	switch config.SelectedProfileID {
	case "", defaultInputProfileID:
		return identityHomography(), false
	case autoInputProfileID:
		return automaticCropHomography(area)
	}

	for _, profile := range config.Profiles {
		if profile.ID != config.SelectedProfileID {
			continue
		}
		calibration, exists := profile.Calibrations[absoluteMouseMode]
		if !exists {
			return identityHomography(), false
		}
		return calibrationHomography(calibration)
	}

	return identityHomography(), false
}

// automaticCropHomography maps the detected content rectangle onto the complete
// HID range, exactly like createAutomaticCropCalibration for the absolute mouse
// mode.
func automaticCropHomography(area contentArea) ([homographySize]float64, bool) {
	left := clamp01(area.left)
	top := clamp01(area.top)
	right := clamp01(area.left + area.width)
	bottom := clamp01(area.top + area.height)

	if right-left <= 0 || bottom-top <= 0 {
		return identityHomography(), false
	}

	pairs := [homographyCorners]homographyPair{
		{source: normalizedPoint{x: left, y: top}, target: normalizedPoint{x: 0, y: 0}},
		{source: normalizedPoint{x: right, y: top}, target: normalizedPoint{x: 1, y: 0}},
		{source: normalizedPoint{x: right, y: bottom}, target: normalizedPoint{x: 1, y: 1}},
		{source: normalizedPoint{x: left, y: bottom}, target: normalizedPoint{x: 0, y: 1}},
	}

	return solveHomography(pairs)
}

func calibrationHomography(calibration proto.InputProfileCalibration) ([homographySize]float64, bool) {
	cornerOrder := [homographyCorners]string{
		calibrationCornerTopLeft,
		calibrationCornerTopRight,
		calibrationCornerBottomRight,
		calibrationCornerBottomLeft,
	}

	pairs := [homographyCorners]homographyPair{}
	for index, corner := range cornerOrder {
		point, exists := calibration.Points[corner]
		if !exists {
			return identityHomography(), false
		}
		pairs[index] = homographyPair{
			source: normalizedPoint{x: point.Source.X, y: point.Source.Y},
			target: normalizedPoint{x: point.Target.X, y: point.Target.Y},
		}
	}

	return solveHomography(pairs)
}

type homographyPair struct {
	source normalizedPoint
	target normalizedPoint
}

// solveHomography solves the eight projective coefficients the same way
// getHomography does in the browser.
func solveHomography(pairs [homographyCorners]homographyPair) ([homographySize]float64, bool) {
	matrix := make([][]float64, 0, homographySize)
	values := make([]float64, 0, homographySize)

	for _, pair := range pairs {
		source := pair.source
		target := pair.target
		matrix = append(matrix,
			[]float64{source.x, source.y, 1, 0, 0, 0, -target.x * source.x, -target.x * source.y},
			[]float64{0, 0, 0, source.x, source.y, 1, -target.y * source.x, -target.y * source.y},
		)
		values = append(values, target.x, target.y)
	}

	solution, ok := solveLinearSystem(matrix, values)
	if !ok {
		return identityHomography(), false
	}

	var homography [homographySize]float64
	for index := range solution {
		if math.IsNaN(solution[index]) || math.IsInf(solution[index], 0) {
			return identityHomography(), false
		}
		homography[index] = solution[index]
	}
	return homography, true
}

func solveLinearSystem(coefficients [][]float64, values []float64) ([]float64, bool) {
	size := len(values)
	rows := make([][]float64, 0, size)
	for index, row := range coefficients {
		extended := make([]float64, 0, size+1)
		extended = append(extended, row...)
		extended = append(extended, values[index])
		rows = append(rows, extended)
	}

	for column := 0; column < size; column++ {
		pivotRow := column
		for rowIndex := column + 1; rowIndex < size; rowIndex++ {
			if math.Abs(rows[rowIndex][column]) > math.Abs(rows[pivotRow][column]) {
				pivotRow = rowIndex
			}
		}
		if math.Abs(rows[pivotRow][column]) < homographyPivotCutoff {
			return nil, false
		}

		rows[column], rows[pivotRow] = rows[pivotRow], rows[column]
		pivot := rows[column][column]
		for index := column; index <= size; index++ {
			rows[column][index] /= pivot
		}

		for rowIndex := 0; rowIndex < size; rowIndex++ {
			if rowIndex == column {
				continue
			}
			factor := rows[rowIndex][column]
			for index := column; index <= size; index++ {
				rows[rowIndex][index] -= factor * rows[column][index]
			}
		}
	}

	solution := make([]float64, size)
	for index := 0; index < size; index++ {
		solution[index] = rows[index][size]
	}
	return solution, true
}

func applyHomography(matrix [homographySize]float64, point normalizedPoint) (float64, float64) {
	denominator := matrix[6]*point.x + matrix[7]*point.y + 1
	if math.IsNaN(denominator) || math.IsInf(denominator, 0) || math.Abs(denominator) < homographyPivotCutoff {
		return clamp01(point.x), clamp01(point.y)
	}

	return clamp01((matrix[0]*point.x + matrix[1]*point.y + matrix[2]) / denominator),
		clamp01((matrix[3]*point.x + matrix[4]*point.y + matrix[5]) / denominator)
}

func identityHomography() [homographySize]float64 {
	return [homographySize]float64{1, 0, 0, 0, 1, 0, 0, 0}
}
