package main

import (
	"os"
	"sync/atomic"

	"github.com/charmbracelet/x/ansi"
)

type cursorAnchorState struct {
	enabled atomic.Bool
	row     atomic.Int64
	col     atomic.Int64
}

var tuiCursorAnchor cursorAnchorState

type tuiViewCacheState struct {
	valid          bool
	view           string
	inputActive    bool
	inputLine      string
	inputWidth     int
	inputRow       int
	inputBaseCol   int
	inputCursorCol int
}

var (
	tuiCachedView       tuiViewCacheState
	tuiRenderCache      tuiViewCacheState
	writeTUIInputCursor = moveTUIInputCursor
)

type cursorAwareOutput struct {
	file *os.File
}

func (w cursorAwareOutput) Read(p []byte) (int, error) {
	return w.file.Read(p)
}

func (w cursorAwareOutput) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil || !tuiCursorAnchor.enabled.Load() {
		return n, err
	}

	row := int(tuiCursorAnchor.row.Load())
	col := int(tuiCursorAnchor.col.Load())
	if row > 0 && col > 0 {
		_, _ = w.file.WriteString(ansi.ShowCursor + ansi.CursorPosition(col, row))
	}
	return n, nil
}

func (w cursorAwareOutput) Close() error {
	return nil
}

func (w cursorAwareOutput) Fd() uintptr {
	return w.file.Fd()
}

func setTUIInputCursor(row int, col int) {
	if row <= 0 || col <= 0 {
		clearTUIInputCursor()
		return
	}
	tuiCursorAnchor.row.Store(int64(row))
	tuiCursorAnchor.col.Store(int64(col))
	tuiCursorAnchor.enabled.Store(true)
}

func clearTUIInputCursor() {
	tuiCursorAnchor.enabled.Store(false)
}

func moveTUIInputCursor(row int, col int) {
	setTUIInputCursor(row, col)
	if row > 0 && col > 0 {
		_, _ = os.Stdout.WriteString(ansi.ShowCursor + ansi.CursorPosition(col, row))
	}
}

func beginTUIViewRender() {
	tuiRenderCache = tuiViewCacheState{}
}

func recordTUIViewInput(inputLine string, inputWidth int, row int, baseCol int, cursorCol int) {
	tuiRenderCache.inputActive = true
	tuiRenderCache.inputLine = inputLine
	tuiRenderCache.inputWidth = inputWidth
	tuiRenderCache.inputRow = row
	tuiRenderCache.inputBaseCol = baseCol
	tuiRenderCache.inputCursorCol = cursorCol
}

func finishTUIViewRender(view string) {
	tuiRenderCache.valid = true
	tuiRenderCache.view = view
	tuiCachedView = tuiRenderCache
}

func cachedTUIView() (string, bool) {
	if !tuiCachedView.valid {
		return "", false
	}
	return tuiCachedView.view, true
}
