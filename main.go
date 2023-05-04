package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/ajdnik/imghash"
	"github.com/ajdnik/imghash/hashtype"
	"github.com/ajdnik/imghash/similarity"
	bolt "go.etcd.io/bbolt"
)

//imagedata struct, json serializable
type ImageData struct {
	Hash hashtype.Binary
	InsertedAt time.Time	
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

func hashImage(path string) (hashtype.Binary, error) {
	// Open image file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
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
			e := ImageData{Hash: h1, InsertedAt: time.Now()}
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

func findSimilarImgs(db *bolt.DB, h1 hashtype.Binary, limitdiff similarity.Distance) (map[similarity.Distance][]ImageData, error) {
	//var similarImgs map[similarity.Distance][]ImageData
	similarImgs := make(map[similarity.Distance][]ImageData)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("images"))
		allhashes := b.Get([]byte("allhashes"))
		var all []hashtype.Binary
		err := json.Unmarshal(allhashes, &all)
		if err != nil {
			return err
		}
		for _, h2 := range all {
			d := diff(h1, h2)
			if d <= limitdiff {
				var similar []ImageData
				//if similarImgs[d] exists, append to it, else create it
				similar = similarImgs[d]
				if similar == nil {
					similarImgs[d] = []ImageData{}
					similar = []ImageData{}
				}
				img := b.Get(h2)
				var e ImageData
				err := json.Unmarshal(img, &e)
				if err != nil {
					return err
				}
				similarImgs[d] = append(similar, e)
			}
		}
		return nil
	})
	return similarImgs, err
}

func main() {
	db, err := bolt.Open("images.bolt", 0666, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	for {
		fmt.Print("Enter image path: ")
		var path string
		fmt.Scanln(&path)
		hash, err := hashImage(path)
		if err != nil {
			panic(err)
		}
		err = insertInDB(hash, db)
		if err != nil {
			panic(err)
		}
		similarImgs, err := findSimilarImgs(db, hash, 10) //one is the image itself
		if err != nil {
			panic(err)
		}
		for d, imgs := range similarImgs {
			if d == 0 {
				continue
			}
			fmt.Printf("Found %v similar images with %v%% difference\n", len(imgs), d)
		}
		if len(similarImgs) == 1 {
			fmt.Println("No similar images found")
		}
	}
}