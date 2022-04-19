package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/devnull-twitch/godot-delivery-endpoint/pkg/bundler"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Platform string

var platforms []Platform = []Platform{"Linux/X11", "Windows Desktop"}

// server application to ( maybe package ) and provide downloads for pck
func main() {
	godotenv.Load(".env.yaml")

	pckStorageDir := filepath.Join(os.Getenv("STORAGE_PATH"), "pcks")
	if _, err := os.Stat(pckStorageDir); err != nil {
		err := os.Mkdir(pckStorageDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	cache := make(map[Platform][]*bundler.PckMetadata)

	r := gin.Default()
	baseGroup := r.Group("godot-delivery")
	{
		baseGroup.POST("/add-repository", func(c *gin.Context) {
			authUser, authPw, hasBasic := c.Request.BasicAuth()
			if !hasBasic || authUser != "manager" || authPw != os.Getenv("AUTH_PW") {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			payload := &struct {
				URL string `json:"repository"`
			}{}
			if err := c.BindJSON(payload); err != nil || payload.URL == "" {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			// TODO maybe put this in a goroutine so it doesnt block the http handler thread
			// what is the nginx default timeout?

			metadataList := bundler.BuildPck(payload.URL)
			for _, metadata := range metadataList {
				if _, ok := cache[Platform(metadata.Platform)]; !ok {
					cache[Platform(metadata.Platform)] = []*bundler.PckMetadata{}
				}

				cache[Platform(metadata.Platform)] = append(cache[Platform(metadata.Platform)], metadata)
			}

			// TODO BONUS also write that to file as JSON backup ...
		})
		baseGroup.GET("/next-game", func(c *gin.Context) {
			currentStr, currentGiven := c.GetQuery("current")
			if !currentGiven {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			platformStr := c.Query("platform")
			userPlatform := Platform(platformStr)
			if !validPlatform(userPlatform) {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			currentIndex, err := strconv.Atoi(currentStr)
			if err != nil {
				logrus.WithError(err).Warn("unable to convert current query to integer")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if len(cache[userPlatform]) < currentIndex+1 {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			url := path.Join(os.Getenv("BASE_URL"), "download", cache[userPlatform][currentIndex+1].Filename)
			c.Data(http.StatusOK, "text/plain", []byte(url))
		})
		baseGroup.Static("/download", filepath.Join(os.Getenv("STORAGE_PATH"), "pcks"))
	}
}

func validPlatform(platformStr Platform) bool {
	for _, testPlatform := range platforms {
		if platformStr == testPlatform {
			return true
		}
	}

	return false
}
