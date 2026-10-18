package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestShaderCompiles(t *testing.T) {
	if _, err := ebiten.NewShader(crtSrc); err != nil {
		t.Fatal(err)
	}
}
