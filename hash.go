package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ajdnik/imghash"
	"github.com/ajdnik/imghash/similarity"
	bolt "go.etcd.io/bbolt"
)

var hashes = []string{
	"average",
	"difference",
	"median",
	"colormoment",
	"marrhildreth",
	"blockmean",
	"phash",
	"radialvariance",
}

var errInvalidHash = errors.New("invalid hash type, send a GET request to / to see all available hashes")

func ListContains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

func GetImageData(imgreader io.Reader) (ImageData, error) {
	img, _, err := image.Decode(imgreader)
	if err != nil {
		return ImageData{}, err
	}
	averageHash := imghash.NewAverage()
	differenceHash := imghash.NewDifference()
	medianHash := imghash.NewMedian()
	colorMomentHash := imghash.NewColorMoment()
	marrHildrethHash := imghash.NewMarrHildreth()
	blockMeanHash := imghash.NewBlockMean()
	pHash := imghash.NewPHash()
	radialVarianceHash := imghash.NewRadialVariance()
	t := time.Now()
	id := base64.URLEncoding.EncodeToString([]byte(strconv.FormatInt(t.UnixNano(), 10)))
	return ImageData{
		AverageHash:        averageHash.Calculate(img),
		DifferenceHash:     differenceHash.Calculate(img),
		MedianHash:         medianHash.Calculate(img),
		ColorMomentHash:    colorMomentHash.Calculate(img),
		MarrHildrethHash:   marrHildrethHash.Calculate(img),
		BlockMeanHash:      blockMeanHash.Calculate(img),
		PHash:              pHash.Calculate(img),
		RadialVarianceHash: radialVarianceHash.Calculate(img),
		InsertedAt:         t,
		ImageID:            id,
	}, nil
}

func FindSimilarImages(data ImageData, hash string, maxdistance float64) (map[string][]ImageData, error) {
	hash = strings.ToLower(hash)
	if !ListContains(hashes, hash) {
		return nil, errInvalidHash
	}
	var similarImages = make(map[string][]ImageData)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("images"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var imgdata ImageData
			err := json.Unmarshal(v, &imgdata)
			if err != nil {
				return err
			}
			var difference similarity.Distance
			switch hash {
				case "average":
					difference = similarity.Hamming(data.AverageHash, imgdata.AverageHash)
				case "difference":
					difference = similarity.Hamming(data.DifferenceHash, imgdata.DifferenceHash)
				case "median":
					difference = similarity.Hamming(data.MedianHash, imgdata.MedianHash)
				case "colormoment":
					difference = similarity.L2Float64(data.ColorMomentHash, imgdata.ColorMomentHash)
				case "marrhildreth":
					difference = similarity.Hamming(data.MarrHildrethHash, imgdata.MarrHildrethHash)
				case "blockmean":
					difference = similarity.Hamming(data.BlockMeanHash, imgdata.BlockMeanHash)
				case "phash":
					difference = similarity.Hamming(data.PHash, imgdata.PHash)
				case "radialvariance":
					difference = similarity.L2UInt8(data.RadialVarianceHash, imgdata.RadialVarianceHash)
			}
			//if difference <= maxdistance image is similar
			//if similarImages[difference] is nil, create a new slice
			strdiff := fmt.Sprintf("%f", difference)
			if (float64(difference) <= maxdistance) && (imgdata.ImageID != data.ImageID) {
				if similarImages[strdiff] == nil {
					similarImages[strdiff] = make([]ImageData, 0)
				}
				similarImages[strdiff] = append(similarImages[strdiff], imgdata)
			}
		}
		return nil

	})
	if err != nil {
		return nil, err
	}
	return similarImages, nil
}