package gomidicreator

import (
	"bytes"
	"encoding/binary"
	"os"
	"fmt"
)

const (
	TICKSPERBEAT = 128
)

type MIDIFile struct {
	header MIDIHeader
	tracks []MIDITrack
}

type MIDIHeader struct {
	numTracks uint16
}

type MIDITrack struct {
	trackno    int
	events     []interface{}
	midiEvents []interface{}
}

type Note struct {
	time     int
	pitch    int
	duration int
	volume   int
}

type TrackName struct {
	time      int
	trackName string
}

type Tempo struct {
	time  int
	tempo int
}

type ProgramChange struct {
	time          int
	programNumber int
}

func (header MIDIHeader) WriteFile(fd *os.File) error {
	var err error

	err = binary.Write(fd, binary.BigEndian, []byte("MThd"))
	if err != nil {
		return err
	}

	err = binary.Write(fd, binary.BigEndian, uint32(6)) // size
	if err != nil {
		return err
	}

	err = binary.Write(fd, binary.BigEndian, uint16(1)) // type
	if err != nil {
		return err
	}

	err = binary.Write(fd, binary.BigEndian, header.numTracks)
	if err != nil {
		return err
	}

	err = binary.Write(fd, binary.BigEndian, uint16(TICKSPERBEAT))
	if err != nil {
		return err
	}

	return nil
}

type MIDINoteOnOff struct {
	time     uint32
	channel  uint8
	number   uint8
	velocity uint8
}

type MIDINoteOn MIDINoteOnOff
type MIDINoteOff MIDINoteOnOff

func (event MIDINoteOn) Pack(buf *bytes.Buffer) {
	code := 0x9<<4 | event.channel
	buf.Write(WriteVarInt(event.time))
	binary.Write(buf, binary.BigEndian, code)
	binary.Write(buf, binary.BigEndian, event.number)
	binary.Write(buf, binary.BigEndian, event.velocity)
	fmt.Println("packed NoteOn event, buffer size is now", buf.Len(), "bytes")
}

func (event MIDINoteOff) Pack(buf *bytes.Buffer) {
	code := 0x9<<4 | event.channel
	buf.Write(WriteVarInt(event.time))
	binary.Write(buf, binary.BigEndian, code)
	binary.Write(buf, binary.BigEndian, event.number)
	binary.Write(buf, binary.BigEndian, event.velocity)
	fmt.Println("packed NoteOff event, buffer size is now", buf.Len(), "bytes")
}

type MIDITempo struct {
	time  uint32
	tempo uint32
}

func (event MIDITempo) Pack(buf *bytes.Buffer) {
	buf.Write(WriteVarInt(event.time))
	binary.Write(buf, binary.BigEndian, 0xFF)
	binary.Write(buf, binary.BigEndian, 0x51)
	binary.Write(buf, binary.BigEndian, 3)
	four_byte_buf := WriteVarInt(event.tempo)
	buf.Write(four_byte_buf[1:])
	fmt.Println("packed Tempo event, buffer size is now", buf.Len(), "bytes")
}

func WriteVarInt(number uint32) []byte {
	buf := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		buf[i] = byte(number & 0x7F)
		number >>= 7
		if number > 0 {
			buf[i] |= 0x80
		}
	}
	return buf
}

func (track MIDITrack) WriteEventsToBuf(buf *bytes.Buffer) error {
	fmt.Println("writing", len(track.midiEvents), "events to buffer...")
	for _, event := range track.midiEvents {
		switch event := event.(type) {
		case MIDINoteOn:
			event.Pack(buf)
		case MIDINoteOff:
			event.Pack(buf)
		case MIDITempo:
			event.Pack(buf)
		default:
			panic("unknown type")
		}
	}

	return nil
}

func (track *MIDITrack) PrepareMIDIEvents() int {
	for _, event := range track.events {
		switch event := event.(type) {
		case Note:
			track.midiEvents = append(track.midiEvents,
				MIDINoteOn{
					time:     uint32(event.time * TICKSPERBEAT),
					channel:  uint8(track.trackno),
					number:   uint8(event.pitch),
					velocity: uint8(event.volume)})
			track.midiEvents = append(track.midiEvents,
				MIDINoteOff{
					time:     uint32((event.time + event.duration) * TICKSPERBEAT),
					channel:  uint8(track.trackno),
					number:   uint8(event.pitch),
					velocity: uint8(event.volume)})
		case Tempo:
			track.midiEvents = append(track.midiEvents,
				MIDITempo{
					time:  uint32(event.time * TICKSPERBEAT),
					tempo: uint32(event.tempo)})
		case ProgramChange:
		case TrackName:
		default:
			panic("unknown")
		}
	}

	return len(track.midiEvents)
}

func (track MIDITrack) WriteFile(fd *os.File) error {

	fmt.Println("writing track #", track.trackno)

	midiEvents := track.PrepareMIDIEvents()
	fmt.Println("prepared", midiEvents, "midi events")

	var buf bytes.Buffer
	track.WriteEventsToBuf(&buf)

	fmt.Println("prepared buffer with length", buf.Len())

	/* write end of track */
	buf.Write([]byte{0xFF, 0x2F, 0x00})

	fmt.Println("prepared buffer with length", buf.Len())

	buf.WriteTo(fd)

	return nil
}

func NewMIDIFile(tracks uint16) MIDIFile {
	file := MIDIFile{}

	file.header = MIDIHeader{
		numTracks: tracks,
	}

	file.tracks = make([]MIDITrack, tracks)
	for i, track := range file.tracks {
		track.trackno = i
	}

	return file
}

func (midi MIDIFile) AddNote(track uint16, pitch, time, duration, volume int) {
	midi.tracks[track].events = append(midi.tracks[track].events, Note{
		pitch:    pitch,
		time:     time,
		duration: duration,
		volume:   volume,
	})
}

func (midi MIDIFile) AddTrackName(track uint16, time int, trackName string) {
	midi.tracks[track].events = append(midi.tracks[track].events, TrackName{
		time:      time,
		trackName: trackName,
	})
}

func (midi MIDIFile) SetTempo(track uint16, time, tempo int) {
	midi.tracks[track].events = append(midi.tracks[track].events, Tempo{
		time:  time,
		tempo: 60000000 / tempo,
	})
}

func (midi MIDIFile) SetProgramChange(track uint16, time, program int) {
	midi.tracks[track].events = append(midi.tracks[track].events, ProgramChange{
		time:          time,
		programNumber: program,
	})
}

func (midi MIDIFile) WriteFile(fd *os.File) error {
	var err error

	err = midi.header.WriteFile(fd)
	if err != nil {
		return err
	}

	for _, track := range midi.tracks {
		err = track.WriteFile(fd)
		if err != nil {
			return err
		}
	}

	return nil
}
