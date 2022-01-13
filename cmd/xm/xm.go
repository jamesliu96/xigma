package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"

	"github.com/jamesliu96/xigma"
	"github.com/jamesliu96/xp"
)

const app = "xm"

var (
	gitTag = "*"
	gitRev = "*"
)

const DIRECTIVE_SERVER = "s"
const DIRECTIVE_CLIENT = "c"

func usage() {
	fmt.Fprintf(os.Stderr, "%s %s (%s)\nusage: %s %s <addr> [dir] # server\n       %s %s <uri>        # client\n", app, gitTag, gitRev[:int(math.Min(float64(len(gitRev)), 7))], app, DIRECTIVE_SERVER, app, DIRECTIVE_CLIENT)
}

const HEADER_SERVER = "x-xigma-server"
const HEADER_CLIENT = "x-xigma-client"

func main() {
	if len(os.Args) < 3 {
		usage()
		return
	}
	directive := os.Args[1]
	addr := os.Args[2]
	if directive == DIRECTIVE_SERVER {
		http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			clientPubString := r.Header.Get(HEADER_CLIENT)
			if len(clientPubString) == 0 {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			clientPub, err := hex.DecodeString(clientPubString)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			serverPriv, serverPub, err := xp.P()
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			rw.Header().Set(HEADER_SERVER, hex.EncodeToString(serverPub))
			shared, err := xp.X(serverPriv, clientPub)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			dir := "."
			if len(os.Args) > 3 {
				dir = os.Args[3]
			}
			f, err := os.Open(dir + r.URL.Path)
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
			log.Println(clientPubString, f.Name())
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
				buf := new(bytes.Buffer)
				for _, de := range des {
					suffix := ""
					if de.IsDir() {
						suffix = "/"
					}
					buf.WriteString(fmt.Sprintf("%s%s\n", de.Name(), suffix))
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
			return
		}
		clientPriv, clientPub, err := xp.P()
		if err != nil {
			log.Fatalln(err)
			return
		}
		req.Header.Set(HEADER_CLIENT, hex.EncodeToString(clientPub))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err)
			return
		}
		if resp.StatusCode == http.StatusOK {
			serverPubString := resp.Header.Get(HEADER_SERVER)
			serverPub, err := hex.DecodeString(serverPubString)
			if err != nil {
				log.Fatalln(err)
				return
			}
			shared, err := xp.X(clientPriv, serverPub)
			if err != nil {
				log.Fatalln(err)
				return
			}
			if err := xigma.Decrypt(resp.Body, os.Stdout, shared); err != nil {
				log.Fatalln(err)
			}
		}
	} else {
		usage()
	}
}
