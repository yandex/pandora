// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/SlyMarbo/spdy/common"
)

type SETTINGS struct {
	Flags    common.Flags
	Settings common.Settings
}

func (frame *SETTINGS) Add(Flags common.Flags, id uint32, value uint32) {
	frame.Settings[id] = &common.Setting{Flags, id, value}
}

func (frame *SETTINGS) Compress(comp common.Compressor) error {
	return nil
}

func (frame *SETTINGS) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *SETTINGS) Name() string {
	return "SETTINGS"
}

func (frame *SETTINGS) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 12)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _SETTINGS, common.FLAG_SETTINGS_CLEAR_SETTINGS)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length < 4 {
		return c.N, common.IncorrectDataLength(length, 8)
	} else if length > common.MAX_FRAME_SIZE-8 {
		return c.N, common.FrameTooLarge
	}

	// Check size.
	numSettings := int(common.BytesToUint32(data[8:12]))
	if length != 4+(8*numSettings) {
		return c.N, common.IncorrectDataLength(length, 4+(8*numSettings))
	}

	// Read in data.
	settings, err := common.ReadExactly(&c, 8*numSettings)
	if err != nil {
		return c.N, err
	}

	frame.Flags = common.Flags(data[4])
	frame.Settings = make(common.Settings)
	for i := 0; i < numSettings; i++ {
		j := i * 8
		setting := decodeSetting(settings[j:])
		if setting == nil {
			return c.N, errors.New("Error: Failed to parse settings.")
		}
		frame.Settings[setting.ID] = setting
	}

	return c.N, nil
}

func (frame *SETTINGS) String() string {
	buf := new(bytes.Buffer)
	flags := ""
	if frame.Flags.CLEAR_SETTINGS() {
		flags += " FLAG_SETTINGS_CLEAR_SETTINGS"
	}
	if flags == "" {
		flags = "[NONE]"
	} else {
		flags = flags[1:]
	}

	buf.WriteString("SETTINGS {\n\t")
	buf.WriteString(fmt.Sprintf("Version:              2\n\t"))
	buf.WriteString(fmt.Sprintf("Flags:                %s\n\t", flags))
	buf.WriteString(fmt.Sprintf("Settings:\n"))
	settings := frame.Settings.Settings()
	for _, setting := range settings {
		buf.WriteString("\t\t" + setting.String() + "\n")
	}
	buf.WriteString("}\n")

	return buf.String()
}

func (frame *SETTINGS) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	settings := encodeSettings(frame.Settings)
	numSettings := uint32(len(frame.Settings))
	length := 4 + len(settings)
	out := make([]byte, 12)

	out[0] = 128                     // Control bit and Version
	out[1] = 2                       // Version
	out[2] = 0                       // Type
	out[3] = 4                       // Type
	out[4] = byte(frame.Flags)       // Flags
	out[5] = byte(length >> 16)      // Length
	out[6] = byte(length >> 8)       // Length
	out[7] = byte(length)            // Length
	out[8] = byte(numSettings >> 24) // Number of Entries
	out[9] = byte(numSettings >> 16) // Number of Entries
	out[10] = byte(numSettings >> 8) // Number of Entries
	out[11] = byte(numSettings)      // Number of Entries

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	err = common.WriteExactly(&c, settings)
	if err != nil {
		return c.N, err
	}

	return c.N, nil
}

func decodeSetting(data []byte) *common.Setting {
	if len(data) < 8 {
		return nil
	}

	setting := new(common.Setting)
	setting.ID = common.BytesToUint24Reverse(data[0:]) // Might need to reverse this.
	setting.Flags = common.Flags(data[3])
	setting.Value = common.BytesToUint32(data[4:])

	return setting
}

func encodeSettings(s common.Settings) []byte {
	if len(s) == 0 {
		return []byte{}
	}

	ids := make([]int, 0, len(s))
	for id := range s {
		ids = append(ids, int(id))
	}

	sort.Sort(sort.IntSlice(ids))

	out := make([]byte, 8*len(s))

	offset := 0
	for _, id := range ids {
		setting := s[uint32(id)]
		out[offset] = byte(setting.ID)         // Might need to reverse this.
		out[offset+1] = byte(setting.ID >> 8)  // Might need to reverse this.
		out[offset+2] = byte(setting.ID >> 16) // Might need to reverse this.
		out[offset+3] = byte(setting.Flags)
		out[offset+4] = byte(setting.Value >> 24)
		out[offset+5] = byte(setting.Value >> 16)
		out[offset+6] = byte(setting.Value >> 8)
		out[offset+7] = byte(setting.Value)
		offset += 8
	}

	return out
}
