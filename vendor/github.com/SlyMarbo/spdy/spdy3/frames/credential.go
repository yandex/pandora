// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io"

	"github.com/SlyMarbo/spdy/common"
)

type CREDENTIAL struct {
	Slot         uint16
	Proof        []byte
	Certificates []*x509.Certificate
}

func (frame *CREDENTIAL) Compress(comp common.Compressor) error {
	return nil
}

func (frame *CREDENTIAL) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *CREDENTIAL) Name() string {
	return "CREDENTIAL"
}

func (frame *CREDENTIAL) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 18)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _CREDENTIAL, 0)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length < 6 {
		return c.N, common.IncorrectDataLength(length, 6)
	} else if length > common.MAX_FRAME_SIZE-8 {
		return c.N, common.FrameTooLarge
	}

	// Read in data.
	certs, err := common.ReadExactly(&c, length-10)
	if err != nil {
		return c.N, err
	}

	frame.Slot = common.BytesToUint16(data[8:10])
	proofLen := int(common.BytesToUint32(data[10:14]))
	if proofLen > 0 {
		frame.Proof = data[14 : 14+proofLen]
	} else {
		frame.Proof = []byte{}
	}

	numCerts := 0
	for offset := 0; offset < length-10; {
		offset += int(common.BytesToUint32(certs[offset:offset+4])) + 4
		numCerts++
	}

	frame.Certificates = make([]*x509.Certificate, numCerts)
	for i, offset := 0, 0; offset < length-10; i++ {
		length := int(common.BytesToUint32(certs[offset : offset+4]))
		rawCert := certs[offset+4 : offset+4+length]
		frame.Certificates[i], err = x509.ParseCertificate(rawCert)
		if err != nil {
			return c.N, err
		}
		offset += length + 4
	}

	return c.N, nil
}

func (frame *CREDENTIAL) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString("CREDENTIAL {\n\t")
	buf.WriteString(fmt.Sprintf("Version:              3\n\t"))
	buf.WriteString(fmt.Sprintf("Slot:                 %d\n\t", frame.Slot))
	buf.WriteString(fmt.Sprintf("Proof:                %v\n\t", frame.Proof))
	buf.WriteString(fmt.Sprintf("Certificates:         %v\n}\n", frame.Certificates))

	return buf.String()
}

func (frame *CREDENTIAL) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	proofLength := len(frame.Proof)
	certsLength := 0
	for _, cert := range frame.Certificates {
		certsLength += len(cert.Raw)
	}

	length := 6 + proofLength + certsLength
	out := make([]byte, 14)

	out[0] = 128                      // Control bit and Version
	out[1] = 3                        // Version
	out[2] = 0                        // Type
	out[3] = 10                       // Type
	out[4] = 0                        // common.Flags
	out[5] = byte(length >> 16)       // Length
	out[6] = byte(length >> 8)        // Length
	out[7] = byte(length)             // Length
	out[8] = byte(frame.Slot >> 8)    // Slot
	out[9] = byte(frame.Slot)         // Slot
	out[10] = byte(proofLength >> 24) // Proof Length
	out[11] = byte(proofLength >> 16) // Proof Length
	out[12] = byte(proofLength >> 8)  // Proof Length
	out[13] = byte(proofLength)       // Proof Length

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	if len(frame.Proof) > 0 {
		err = common.WriteExactly(&c, frame.Proof)
		if err != nil {
			return c.N, err
		}
	}

	written := int64(14 + len(frame.Proof))
	for _, cert := range frame.Certificates {
		err = common.WriteExactly(&c, cert.Raw)
		if err != nil {
			return c.N, err
		}
		written += int64(len(cert.Raw))
	}

	return c.N, nil
}
