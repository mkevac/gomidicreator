package main

import (
	"fmt"
	"github.com/mkevac/gomidicreator"
	"github.com/mkevac/gomusic"
)

func main() {
	midifile := gomidicreator.NewMIDIFile(1)
	midifile.AddTrackName(0, 0, "MainTrack")
	midifile.AddNote(0, 1, 1, 1, 1)
	midifile.AddNote(0, 2, 2, 2, 2)
	fmt.Printf("%#v\n", midifile)
}
