package main

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"

	"github.com/jamesliu96/geheim"
	"github.com/jamesliu96/xigma"
)

const app = "xm"

var (
	gitTag = "*"
	gitRev = "*"
)

var fKeyHex = flag.String("x", "", "key `hex` [server: pub, client: priv]")

const DIRECTIVE_SERVER = "s"
const DIRECTIVE_CLIENT = "c"

const HEADER_SERVER = "x-xigma-server"
const HEADER_CLIENT = "x-xigma-client"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\nusage: %s [option]... %s <addr> [dir] # server\n       %s [option]... %s <uri>        # client\noptions:\n", app, gitTag, gitRev[:int(math.Min(float64(len(gitRev)), 7))], app, DIRECTIVE_SERVER, app, DIRECTIVE_CLIENT)
		flag.PrintDefaults()
	}
	if len(os.Args) <= 1 {
		flag.Usage()
		return
	}
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		return
	}
	directive := flag.Arg(0)
	addr := flag.Arg(1)
	var keySet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "x" {
			keySet = true
		}
	})
	var key []byte
	if keySet {
		if k, err := hex.DecodeString(*fKeyHex); err != nil {
			log.Fatalln(err)
			return
		} else {
			key = k
		}
	}
	if directive == DIRECTIVE_SERVER {
		if key != nil {
			log.Println("authorization", hex.EncodeToString(key))
		}
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
			if key != nil {
				if !hmac.Equal(clientPub, key) {
					log.Println("rejected", hex.EncodeToString(clientPub))
					rw.WriteHeader(http.StatusUnauthorized)
					return
				}
			}
			log.Println("resolved", hex.EncodeToString(clientPub))
			serverPriv, serverPub, err := geheim.P()
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			rw.Header().Set(HEADER_SERVER, hex.EncodeToString(serverPub))
			shared, err := geheim.X(serverPriv, clientPub)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			dir := "."
			if flag.NArg() > 2 {
				dir = flag.Arg(2)
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
		log.Println("listening on", addr)
		log.Fatalln(http.ListenAndServe(addr, nil))
	} else if directive == DIRECTIVE_CLIENT {
		req, err := http.NewRequest(http.MethodPost, addr, nil)
		if err != nil {
			log.Fatalln(err)
			return
		}
		var clientPriv, clientPub []byte
		if key != nil {
			if pub, err := geheim.X(key, nil); err != nil {
				log.Fatalln(err)
				return
			} else {
				clientPriv, clientPub = key, pub
			}
		} else {
			if priv, pub, err := geheim.P(); err != nil {
				log.Fatalln(err)
				return
			} else {
				clientPriv, clientPub = priv, pub
			}
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
			shared, err := geheim.X(clientPriv, serverPub)
			if err != nil {
				log.Fatalln(err)
				return
			}
			if err := xigma.Decrypt(resp.Body, os.Stdout, shared); err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Fatalln(resp.Status)
			return
		}
	} else {
		flag.Usage()
	}
}
