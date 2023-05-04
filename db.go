package main

import (
	"encoding/json"
	"time"

	"github.com/ajdnik/imghash/hashtype"
	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB

//imagedata struct, json serializable
type ImageData struct {
	AverageHash 		hashtype.Binary 	`json:"average_hash"`
	DifferenceHash 		hashtype.Binary 	`json:"difference_hash"`
	MedianHash 			hashtype.Binary 	`json:"median_hash"`
	ColorMomentHash 	hashtype.Float64 	`json:"color_moment_hash"`
	MarrHildrethHash 	hashtype.Binary 	`json:"marr_hildreth_hash"`
	BlockMeanHash 		hashtype.Binary 	`json:"block_mean_hash"`
	PHash 				hashtype.Binary 	`json:"p_hash"`
	RadialVarianceHash 	hashtype.UInt8	 	`json:"radial_variance_hash"`	
	InsertedAt 			time.Time 	`json:"inserted_at"`
	ImageID 			string 		`json:"image_id"`
}	


func GetImageFromDB(id string) (ImageData, bool, error) {
	var imgdata ImageData
	var found bool
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("images"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(id))
		if v != nil {
			found = true
		} else {
			return nil
		}
		err := json.Unmarshal(v, &imgdata)
		if err != nil {
			return err
		}
		return nil
	})
	return imgdata, found, err
}

func InsertImage(imgdata ImageData) error {//insert image id with it's respective hashes into the database
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("images"))
		if err != nil {
			return err
		}
		//set key id to value imgdata
		jimgdata, err := json.Marshal(imgdata)
		if err != nil {
			return err
		}
		err = b.Put([]byte(imgdata.ImageID), jimgdata)
		if err != nil {
			return err
		}
		return nil
	})
}