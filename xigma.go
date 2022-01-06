package xigma

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/jamesliu96/geheim"
	"golang.org/x/crypto/curve25519"
)

const DefaultSec = 6

func Pair() (private, public []byte, err error) {
	private = make([]byte, curve25519.ScalarSize)
	_, err = rand.Read(private)
	if err != nil {
		return
	}
	public, err = curve25519.X25519(private, curve25519.Basepoint)
	return
}

func Share(ourprivate, theirpublic []byte) (shared []byte, err error) {
	return curve25519.X25519(ourprivate, theirpublic)
}

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
	signed, err := geheim.Encrypt(r, w, pass, geheim.DefaultCipher, geheim.DefaultMode, geheim.DefaultKDF, geheim.DefaultMAC, geheim.DefaultMD, DefaultSec, nil)
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
