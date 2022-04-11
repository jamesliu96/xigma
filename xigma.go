package xigma

import (
	"crypto/hmac"
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/jamesliu96/geheim"
	"golang.org/x/term"
)

const defaultSec = 6

var headersize int64

func init() {
	meta := geheim.NewMeta(geheim.HeaderVersion)
	header, _ := meta.Header()
	headersize = int64(binary.Size(meta) + binary.Size(header))
}

const progressDuration = time.Second

func wrapProgress(r io.Reader, w *os.File, total int64) (wrapped io.Reader, done chan<- struct{}) {
	wrapped = r
	d := make(chan struct{})
	if !term.IsTerminal(int(w.Fd())) {
		p := &geheim.ProgressWriter{TotalBytes: total}
		go p.Progress(d, progressDuration)
		wrapped = io.TeeReader(r, p)
		done = d
	}
	return
}

func doneProgress(done chan<- struct{}) {
	if done != nil {
		done <- struct{}{}
	}
}

func Encrypt(r io.Reader, w io.Writer, pass []byte, filesize int64) (err error) {
	size := headersize + filesize
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

func Decrypt(r io.Reader, w *os.File, pass []byte) (err error) {
	size := new(int64)
	err = binary.Read(r, binary.BigEndian, size)
	if err != nil {
		return
	}
	wrapped, done := wrapProgress(io.LimitReader(r, *size), w, *size)
	signed, err := geheim.Decrypt(wrapped, w, pass, nil)
	doneProgress(done)
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
