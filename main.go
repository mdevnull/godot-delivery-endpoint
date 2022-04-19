package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/devnull-twitch/godot-delivery-endpoint/pkg/bundler"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Platform string

var platforms []Platform = []Platform{"linuxx11", "windowsdesktop"}

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
	cacheFile := filepath.Join(os.Getenv("STORAGE_PATH"), "cache.json")

	if _, err := os.Stat(cacheFile); err == nil {
		cacheBuf, _ := ioutil.ReadFile(cacheFile)
		if len(cacheBuf) > 0 {
			json.Unmarshal(cacheBuf, &cache)
			logrus.Info("restored cache from file")
		}
	}

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
			if len(metadataList) <= 0 {
				logrus.Info("no exports found to add to metadata list")
				c.Status(http.StatusNoContent)
				return
			}

			gameName := metadataList[0].Gamename
			if gameName == "" {
				logrus.Info("empty game name. aborting")
				c.Status(http.StatusBadRequest)
				return
			}

			// remove all old entries for the same game name
			for platformKey := range cache {
				cleaned := make([]*bundler.PckMetadata, 0, len(cache[platformKey]))
				for _, entry := range cache[platformKey] {
					if entry.Gamename != gameName {
						cleaned = append(cleaned, entry)
					}
				}
			}

			for _, metadata := range metadataList {
				if _, ok := cache[Platform(metadata.Platform)]; !ok {
					cache[Platform(metadata.Platform)] = []*bundler.PckMetadata{}
				}

				cache[Platform(metadata.Platform)] = append(cache[Platform(metadata.Platform)], metadata)
				logrus.WithFields(logrus.Fields{
					"platform": metadata.Platform,
					"pck_file": metadata.Filename,
				}).Info("added metadata to cache")
			}

			cacheBuf, err := json.Marshal(&cache)
			if err == nil {
				ioutil.WriteFile(cacheFile, cacheBuf, os.ModePerm)
				logrus.Info("write cache to file")
			}

			c.Status(http.StatusNoContent)
		})
		baseGroup.GET("/next-game", func(c *gin.Context) {
			platformStr := c.Query("platform")
			userPlatform := Platform(platformStr)
			if !validPlatform(userPlatform) {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			nextIndex := findNextIndex(cache[userPlatform], c.Query("gamename"))
			nextMetaData := cache[userPlatform][nextIndex]
			url := fmt.Sprintf("%s%s", os.Getenv("BASE_URL"), path.Join("godot-delivery/download", nextMetaData.Filename))
			c.JSON(http.StatusOK, &struct {
				DownloadURL string `json:"download_url"`
				GameName    string `json:"game_name"`
				MainScene   string `json:"main_scene"`
			}{
				DownloadURL: url,
				GameName:    nextMetaData.Gamename,
				MainScene:   nextMetaData.MainScene,
			})
		})
		baseGroup.Static("/download", filepath.Join(os.Getenv("STORAGE_PATH"), "pcks"))
	}

	if err := r.Run(os.Getenv("WEBADDRESS")); err != nil {
		log.Fatal(err)
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

func findNextIndex(cache []*bundler.PckMetadata, gameName string) int {
	for index, cacheEntry := range cache {
		if cacheEntry.Gamename == gameName {
			if index < len(cache)-1 {
				return index + 1
			}

			return 0
		}
	}

	return 0
}
