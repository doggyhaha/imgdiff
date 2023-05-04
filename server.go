package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ajdnik/imghash/similarity"
	"github.com/gin-gonic/gin"
	bolt "go.etcd.io/bbolt"
)

type Response struct {
	Ok 			bool 	`json:"ok"`
	Error		string 	`json:"error"`
	Response 	interface{} `json:"response"`
}

// /similarities
func SimilaritiesHandler(c *gin.Context) {
	// Get file from request
	//c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(30<<20))
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: "No file in request",
			Response: nil,
		})
		return
	}
	file, err := files[0].Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	defer file.Close()
	//limit file size
	hash, err := hashImage(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	err = insertInDB(hash, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	//get dist from query params
	strdist := c.Query("dist")
	if strdist == "" {
		strdist = "10"
	}
	dist, err := strconv.Atoi(strdist)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	
	similarImgs, err := findSimilarImgs(db, hash, similarity.Distance(dist)) //one is the image itself
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Ok: false,
			Error: err.Error(),
			Response: nil,
		})
		return
	}
	for d, imgs := range similarImgs {
		if d == "0" {
			delete(similarImgs, d)
			continue
		}
		fmt.Printf("Found %v similar images with %v%% difference\n", len(imgs), d)
	}
	if len(similarImgs) == 0 {
		fmt.Println("No similar images found")
		//{"ok": true, "similarities": null}
		c.JSON(http.StatusOK, Response{
			Ok: true,
			Error: "",
			Response: map[string]interface{}{
				"similarities": nil,
			},
		})
	} else {
		//{"ok": true, "similarities": json serialized map}
		//fmt.Println(similarImgs["1"][0].PHash)
		c.JSON(http.StatusOK, Response{
			Ok: true,
			Error: "",
			Response: similarImgs,
		})
	}
}

func main() {
	var err error
	db, err = bolt.Open("images.bolt", 0666, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	router := gin.Default()
	//gin.SetMode(gin.ReleaseMode)
	router.POST("/similarities", SimilaritiesHandler)
	//router.POST("/diff", DiffHandler)
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"endpoints": []string{"/similarities", "/diff", "/info"},
		})
		},
	)

	router.Run(":6969")
}
