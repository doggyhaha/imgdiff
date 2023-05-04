package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ajdnik/imghash"
	"github.com/ajdnik/imghash/hashtype"
	"github.com/ajdnik/imghash/similarity"
	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB

//imagedata struct, json serializable
type ImageData struct {
	PHash hashtype.Binary 	`json:"phash"`
	InsertedAt time.Time 	`json:"inserted_at"`
}	

func contains(s []hashtype.Binary, e hashtype.Binary) bool {
	for _, a := range s {
		if a.Equal(e) {
			return true
		}
	}
	return false
}

func diff(h1 hashtype.Binary, h2 hashtype.Binary) (similarity.Distance) {
	//fmt.Printf("First hash: %v\n", h1)
	//fmt.Printf("Second hash: %v\n", h2)
	// Compute hash similarity score
	d := similarity.Hamming(h1, h2)
	//fmt.Printf("Hash difference: %v%%\n", d)
	return d
}

func hashImage(file io.Reader) (hashtype.Binary, error) {
	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	h := imghash.NewPHash()
	hash := h.Calculate(img)
	fmt.Printf("Hash: %v\n", hash)
	return hash, nil
}

func insertInDB(h1 hashtype.Binary, db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("images"))
		if err != nil {
			return err
		}
		//get key "allhashes" from bucket "images", if it exists, else create it
		var all []hashtype.Binary
		allhashes := b.Get([]byte("allhashes"))
		if allhashes == nil {
			b.Put([]byte("allhashes"), []byte("[]"))
			allhashes = []byte("[]")
		}
		//unmarshal json into all
		err = json.Unmarshal(allhashes, &all)
		if err != nil {
			return err
		}
		//if h1 is not in all, add it
		if !contains(all, h1) {
			all = append(all, h1)
			//marshal all into alljson
			alljson, err := json.Marshal(all)
			if err != nil {
				return err
			}
			b.Put([]byte("allhashes"), alljson)
		}
		myhash := b.Get(h1)
		if myhash == nil {
			e := ImageData{PHash: h1, InsertedAt: time.Now()}
			fmt.Println(e)
			//marshal json into e
			ejson, err := json.Marshal(e)
			if err != nil {
				return err
			}
			b.Put(h1, ejson)
		} 
		return nil
	},
	)
}

func findSimilarImgs(db *bolt.DB, h1 hashtype.Binary, limitdiff similarity.Distance) (map[string][]ImageData, error) {
	//var similarImgs map[similarity.Distance][]ImageData
	similarImgs := make(map[string][]ImageData)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("images"))
		allhashes := b.Get([]byte("allhashes"))
		var all []hashtype.Binary
		err := json.Unmarshal(allhashes, &all)
		if err != nil {
			return err
		}
		fmt.Println("All hashes: ", len(all))
		for _, h2 := range all {
			d := diff(h1, h2)
			if d <= limitdiff {
				var similar []ImageData
				//if similarImgs[d] exists, append to it, else create it
				std := fmt.Sprint(d)
				similar = similarImgs[std]
				if similar == nil {
					similarImgs[std] = []ImageData{}
					similar = []ImageData{}
				}
				img := b.Get(h2)
				var e ImageData
				err := json.Unmarshal(img, &e)
				if err != nil {
					return err
				}
				similarImgs[std] = append(similar, e)
			}
		}
		return nil
	})
	return similarImgs, err
}

// /diff
func handleDiff(w http.ResponseWriter, r *http.Request) {
	//get two files (max 10MB each)
	defer r.Body.Close()
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		//{"ok": false, "error": "file too large (max 10MB)"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "file too large (max 10MB)"}`)
		return
	}
	//get the similarity tolerance
	tolerance := r.FormValue("tolerance")
	if tolerance == "" {
		tolerance = "10"
	}
	dist, err := strconv.Atoi(tolerance)
	if err != nil {
		//{"ok": false, "error": "tolerance must be an integer"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "tolerance must be an integer"}`)
		return
	}
	//get the files
	file1, _, err := r.FormFile("file1")
	if err != nil {
		//{"ok": false, "error": "file1 not provided"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "file1 not provided"}`)
		return
	}
	defer file1.Close()
	file2, _, err := r.FormFile("file2")
	if err != nil {
		//{"ok": false, "error": "file2 not provided"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "file2 not provided"}`)
		return
	}
	defer file2.Close()
	//get the hashes
	hash1, err := hashImage(file1)
	if err != nil {
		//{"ok": false, "error": "file1 not an image"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "file1 not an image"}`)
		return
	}
	hash2, err := hashImage(file2)
	if err != nil {
		//{"ok": false, "error": "file2 not an image"}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok": false, "error": "file2 not an image"}`)
		return
	}
	//get the difference
	d := diff(hash1, hash2)
	//if the difference is less than the tolerance, return true
	if d <= similarity.Distance(dist) {
		//{"ok": true, "diff": 10, "similar": true}
		fmt.Fprintf(w, `{"ok": true, "diff": %v, "similar": true}`, d)
		return
	} else {
		//{"ok": true, "diff": 10, "similar": false}
		fmt.Fprintf(w, `{"ok": true, "diff": %v, "similar": false}`, d)
		return
	}
}


