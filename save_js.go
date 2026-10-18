//go:build js

package main

import (
	"errors"
	"syscall/js"
)

const canQuit = false

const saveKey = "darkzant-save"

// The browser build keeps its save in localStorage.
func writeSave(b []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("browser storage blocked")
		}
	}()
	js.Global().Get("localStorage").Call("setItem", saveKey, string(b))
	return nil
}

func readSave() (b []byte, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("browser storage blocked")
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", saveKey)
	if v.IsNull() {
		return nil, errors.New("no save")
	}
	return []byte(v.String()), nil
}
