package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func walk(o fyne.CanvasObject, fn func(fyne.CanvasObject)) {
	if o == nil {
		return
	}
	fn(o)
	switch w := o.(type) {
	case *fyne.Container:
		for _, child := range w.Objects {
			walk(child, fn)
		}
	case *Section:
		walk(w.Obj, fn)
	case *container.Scroll:
		walk(w.Content, fn)
	case *container.Clip:
		walk(w.Content, fn)
	case *container.Split:
		walk(w.Leading, fn)
		walk(w.Trailing, fn)
	case *DualPane:
		walk(w.split, fn)
	}
}

func collectText(o fyne.CanvasObject) string {
	var parts []string
	walk(o, func(c fyne.CanvasObject) {
		switch w := c.(type) {
		case *widget.Label:
			if w.Text != "" {
				parts = append(parts, w.Text)
			}
		case *widget.RichText:
			if text := w.String(); text != "" {
				parts = append(parts, text)
			}
		case *widget.Button:
			if w.Text != "" {
				parts = append(parts, w.Text)
			}
		}
	})
	return strings.Join(parts, "\n")
}

func findID(o fyne.CanvasObject, id string) fyne.CanvasObject {
	var found fyne.CanvasObject
	walk(o, func(c fyne.CanvasObject) {
		if found != nil {
			return
		}
		if s, ok := c.(*Section); ok && s.ID == id {
			found = s
		}
	})
	return found
}

func findButton(o fyne.CanvasObject, label string) *widget.Button {
	var found *widget.Button
	walk(o, func(c fyne.CanvasObject) {
		if found != nil {
			return
		}
		if b, ok := c.(*widget.Button); ok && b.Text == label {
			found = b
		}
	})
	return found
}
