package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"NanoKVM-Server/service/hid"
)

const (
	minDragDurationMS     = 100
	defaultDragDurationMS = 300
	maxDragDurationMS     = 1000
	dragStepInterval      = 16 * time.Millisecond
)

func (r *Remote) dragAbsolute(ctx context.Context, start, end normalizedPoint, duration time.Duration) (err error) {
	r.hidMu.Lock()
	defer r.hidMu.Unlock()

	if err := r.checkDragReady(ctx); err != nil {
		return err
	}
	area, _, _, _ := currentContentArea(ctx)
	mapPoint := absolutePointMapper(area)
	routePointer := func(report []byte) error {
		return hid.RouteReliablePointerInput(hid.InternalInputSessionAI, report)
	}
	move := func(point normalizedPoint) error {
		if err := r.checkDragReady(ctx); err != nil {
			return err
		}
		x, y := mapPoint(point.x, point.y)
		return routePointer(buildAbsolutePointerReport(absoluteCoordinate(x), absoluteCoordinate(y), false, false))
	}
	defer func() {
		if releaseErr := r.releaseInput(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("release drag input session: %w", releaseErr))
		}
	}()
	if err := move(start); err != nil {
		return fmt.Errorf("move to drag start: %w", err)
	}
	if err := r.checkDragReady(ctx); err != nil {
		return err
	}

	// Follow Mouse sends absolute position and relative button state on separate
	// HID endpoints. Attempt a button-up even if the press write reports failure.
	defer func() {
		if releaseErr := routePointer(buildRelativeMouseReport(0, 0, 0, 0)); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("release left mouse button: %w", releaseErr))
		}
	}()
	if err := routePointer(buildRelativeMouseReport(0, 0, 1, 0)); err != nil {
		return fmt.Errorf("press left mouse button: %w", err)
	}

	steps := int((duration + dragStepInterval - 1) / dragStepInterval)
	pressedAt := time.Now()
	for step := 1; step <= steps; step++ {
		moveAt := pressedAt.Add(duration * time.Duration(step) / time.Duration(steps))
		if err := waitForDragStep(ctx, moveAt); err != nil {
			return err
		}
		fraction := float64(step) / float64(steps)
		point := normalizedPoint{
			x: (1-fraction)*start.x + fraction*end.x,
			y: (1-fraction)*start.y + fraction*end.y,
		}
		if err := move(point); err != nil {
			return fmt.Errorf("move during drag: %w", err)
		}
	}
	return ctx.Err()
}

func (r *Remote) checkDragReady(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.pointerModeStatus == nil {
		return hid.ErrInputRouterUnavailable
	}
	status := r.pointerModeStatus()
	if status.Mode != hid.PointerModeAbsolute || status.Switching || status.Degraded {
		return fmt.Errorf("drag_mouse requires Follow Mouse with the absolute pointer mode ready (mode %q, switching=%t, degraded=%t)", status.Mode, status.Switching, status.Degraded)
	}
	return nil
}

func waitForDragStep(ctx context.Context, moveAt time.Time) error {
	timer := time.NewTimer(time.Until(moveAt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
