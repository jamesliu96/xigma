package xigma

import (
	"crypto/hmac"
	"encoding/binary"
	"io"

	"github.com/jamesliu96/geheim"
)

const defaultSec = 6

var metasize, headersize int64

func init() {
	meta := geheim.NewMeta(geheim.HeaderVersion)
	header, _ := meta.Header()
	metasize = int64(binary.Size(meta))
	headersize = int64(binary.Size(header))
}

func Encrypt(r io.Reader, w io.Writer, pass []byte, filesize int64) (err error) {
	size := int64(metasize + headersize + filesize)
	err = binary.Write(w, binary.BigEndian, &size)
	if err != nil {
		return
	}
	signed, err := geheim.Encrypt(r, w, pass, geheim.DefaultCipher, geheim.DefaultMode, geheim.DefaultKDF, geheim.DefaultMAC, geheim.DefaultMD, defaultSec, nil)
	if err != nil {
		return
	}
	_, err = w.Write(signed)
	return
}

func Decrypt(r io.Reader, w io.Writer, pass []byte) (err error) {
	size := new(int64)
	err = binary.Read(r, binary.BigEndian, size)
	if err != nil {
		return
	}
	signed, err := geheim.Decrypt(io.LimitReader(r, *size), w, pass, nil)
	if err != nil {
		return
	}
	signex, err := io.ReadAll(r)
	if err != nil {
		return
	}
	if !hmac.Equal(signex, signed) {
		err = geheim.ErrSigVer
	}
	return
}
