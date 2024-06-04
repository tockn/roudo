package roudo_event

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics
#import <CoreGraphics/CoreGraphics.h>
#import <Foundation/Foundation.h>

CGPoint getMouseLocation() {
    CGEventRef event = CGEventCreate(NULL);
    CGPoint cursor = CGEventGetLocation(event);
    CFRelease(event);
    return cursor;
}
*/
import "C"
import (
	"fmt"
	"log/slog"
	"math"
	"time"
)

type MouseEventWatcher struct {
	logger *slog.Logger
}

func (w *MouseEventWatcher) Name() string {
	return "MouseEventWatcher"
}

func (w *MouseEventWatcher) Watch(onEvent func()) error {
	lastPosition := &position{
		x: 0,
		y: 0,
	}
	for {
		time.Sleep(30 * time.Second)
		loc := C.getMouseLocation()
		currentPosition := &position{
			x: float64(loc.x),
			y: float64(loc.y),
		}
		distance := lastPosition.calcDistance(currentPosition)
		w.logger.Debug("kansi mouse event", slog.Float64("currentX", currentPosition.x), slog.Float64("currentY", currentPosition.y), slog.Float64("lastX", lastPosition.x), slog.Float64("lastY", lastPosition.y), slog.Float64("distance", distance))
		if distance > 100 {
			fmt.Println("Mouse moved!")
			onEvent()
			lastPosition = currentPosition
		}
	}
	return nil
}

type position struct {
	x, y float64
}

func (p1 *position) calcDistance(p2 *position) float64 {
	return math.Sqrt(math.Pow(p1.x-p2.x, 2) + math.Pow(p1.y-p2.y, 2))
}
