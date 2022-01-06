package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/jamesliu96/xigma"
	"github.com/jamesliu96/xp"
)

const DIRECTIVE_SERVER = "s"
const DIRECTIVE_CLIENT = "c"

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: xm %s <ADDRESS> [DIRECTORY]\n       xm %s <URL>\n", DIRECTIVE_SERVER, DIRECTIVE_CLIENT)
}

const HEADER_SERVER = "x-xigma-server"
const HEADER_CLIENT = "x-xigma-client"

func main() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}
	directive := os.Args[1]
	addr := os.Args[2]
	if directive == DIRECTIVE_SERVER {
		http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Redirect(rw, r, "https://github.com/jamesliu96/xigma", http.StatusMovedPermanently)
				return
			}
			clientPublicString := r.Header.Get(HEADER_CLIENT)
			if len(clientPublicString) == 0 {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			clientPublic, err := base64.StdEncoding.DecodeString(clientPublicString)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			serverPrivate, serverPublic, err := xp.P()
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			rw.Header().Set(HEADER_SERVER, base64.StdEncoding.EncodeToString(serverPublic))
			shared, err := xp.X(serverPrivate, clientPublic)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			directory := "."
			if len(os.Args) > 3 {
				directory = os.Args[3]
			}
			f, err := os.Open(directory + r.URL.Path)
			if err != nil {
				if os.IsNotExist(err) {
					rw.WriteHeader(http.StatusNotFound)
				} else {
					rw.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			defer (func() {
				if f.Close() != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			})()
			fi, err := f.Stat()
			if err != nil {
				if os.IsNotExist(err) {
					rw.WriteHeader(http.StatusNotFound)
				} else {
					rw.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			if fi.IsDir() {
				des, err := f.ReadDir(-1)
				if err != nil {
					if os.IsNotExist(err) {
						rw.WriteHeader(http.StatusNotFound)
					} else {
						rw.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
				sort.Slice(des, func(i, j int) bool { return des[i].Name() < des[j].Name() })
				var sb strings.Builder
				for _, de := range des {
					suffix := ""
					if de.IsDir() {
						suffix = "/"
					}
					sb.WriteString(fmt.Sprintf("%s%s\n", de.Name(), suffix))
				}
				buf := new(bytes.Buffer)
				if _, err := buf.WriteString(sb.String()); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				if err := xigma.Encrypt(buf, rw, shared, int64(buf.Len())); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				size := fi.Size()
				if size > 0 {
					if err := xigma.Encrypt(f, rw, shared, size); err != nil {
						rw.WriteHeader(http.StatusInternalServerError)
					}
				} else {
					rw.WriteHeader(http.StatusNoContent)
				}
			}
		})
		log.Fatalln(http.ListenAndServe(addr, nil))
	} else if directive == DIRECTIVE_CLIENT {
		req, err := http.NewRequest(http.MethodPost, addr, nil)
		if err != nil {
			log.Fatalln(err)
		}
		clientPrivate, clientPublic, err := xp.P()
		if err != nil {
			log.Fatalln(err)
		}
		req.Header.Set(HEADER_CLIENT, base64.StdEncoding.EncodeToString(clientPublic))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		if resp.StatusCode == http.StatusOK {
			serverPublicString := resp.Header.Get(HEADER_SERVER)
			serverPublic, err := base64.StdEncoding.DecodeString(serverPublicString)
			if err != nil {
				log.Fatalln(err)
			}
			shared, err := xp.X(clientPrivate, serverPublic)
			if err != nil {
				log.Fatalln(err)
			}
			if err := xigma.Decrypt(resp.Body, os.Stdout, shared); err != nil {
				log.Fatalln(err)
			}
		}
	} else {
		printUsage()
	}
}
