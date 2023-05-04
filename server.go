package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	bolt "go.etcd.io/bbolt"
)

type Response struct {
	Ok 			bool 	`json:"ok"`
	Error		string 	`json:"error"`
	Response 	interface{} `json:"response"`
}

// /upload
func UploadHandler(c *gin.Context) { //user uploads image and gets back an ID
	// get image from request
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	// open image
	img, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	defer img.Close()
	// get image data
	imgData, err := GetImageData(img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	err = InsertImage(imgData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	c.JSON(http.StatusOK, Response{
		Ok: true,
		Error: "",
		Response: imgData,
	})
}

// /similarities
func SimilaritiesHandler(c *gin.Context) {
	//get image id from query
	//get hash type from query
	//get max distance from query
	id := c.Query("id")
	hashType := c.Query("hash")
	maxDistance := c.Query("max_distance")
	if id == "" {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: "id is required",
			Response: nil,
		})
		return
	}
	if hashType == "" {
		hashType = "PHash"
	} else if !ListContains(hashes, hashType) {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: errInvalidHash.Error(),
			Response: nil,
		})
		return
	}
	if maxDistance == "" {
		maxDistance = "10"
	}
	//get image data from db
	imgData, found, err := GetImageFromDB(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, Response{
			Ok: false,
			Error: "image ID not found",
			Response: nil,
		})
		return
	}
	//get similar images
	floatDistance, err := strconv.ParseFloat(maxDistance, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	similarImages, err := FindSimilarImages(imgData, hashType, floatDistance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	c.JSON(http.StatusOK, Response{
		Ok: true,
		Error: "",
		Response: similarImages,
	})
}

func main() {
	var err error
	db, err = bolt.Open("images.bolt", 0666, nil) //initialize db globally
	if err != nil {
		panic(err)
	}
	defer db.Close()
	router := gin.Default()
	//gin.SetMode(gin.ReleaseMode)
	router.GET("/similarities", SimilaritiesHandler)
	router.POST("/upload", UploadHandler)
	//router.POST("/diff", DiffHandler)
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"endpoints": []string{"/similarities", "/upload", "/"},
			"hashtypes": hashes,
		})
		},
	)

	router.Run(":6969")
}
