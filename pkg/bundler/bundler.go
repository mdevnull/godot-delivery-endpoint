package bundler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	ini "github.com/ochinchina/go-ini"
	"github.com/sirupsen/logrus"
)

type PckMetadata struct {
	Filename         string `json:"filename"`
	Platform         string `json:"platform"`
	OriginRepository string `json:"origin_repo"`
	Gamename         string `json:"game_name"`
	MainScene        string `json:"main_scene"`
}

var godotBinary string = os.Getenv("GODOT_BINARY")

// TODO maybe build a one thread worker for this? to not get race conditions?? if new entries can be added via HTTP ???
func BuildPck(gitURL string) []*PckMetadata {
	_, err := git.PlainClone("/tmp/foo", false, &git.CloneOptions{
		URL: gitURL,
	})
	if err != nil {
		logrus.WithError(err).Error("unable to clone repository")
		return []*PckMetadata{}
	}
	defer func() {
		_, err := os.Stat("/tmp/foo")
		if err == nil {
			os.RemoveAll("/tmp/foo")
		}
	}()

	listOfIndicationOfCSharp, err := filepath.Glob("/tmp/foo/*.csproj")
	if err != nil {
		panic(err)
	}

	if len(listOfIndicationOfCSharp) > 0 {
		logrus.Info("found csproj... building solution")
		// YO. this is a c sharp godot project. so lets run --build-solution
		cmd := exec.Command(godotBinary, "--build-solution", "--no-window", "-q", "--path", "/tmp/foo")
		cmd.Dir = "/tmp/foo"
		cmd.Run()
	}

	exports := readPresetNames()
	if len(exports) <= 0 {
		logrus.Error("no export preset found")
		return []*PckMetadata{}
	}

	gameName, mainScene := readGamename()
	if gameName == "" {
		return []*PckMetadata{}
	}

	pckFiles := make([]*PckMetadata, 0)
	for _, presetData := range exports {
		exportName := presetData[0]
		platformName := presetData[1]
		logWriter := bytes.NewBuffer([]byte{})

		// godot is here
		// /home/devnull/Downloads/Godot_v3.5-beta3_x11.64
		cmd := exec.Command(godotBinary, "--no-window", "--export-pack", exportName, "/tmp/foo/export.pck")
		cmd.Dir = "/tmp/foo"
		cmd.Stderr = logWriter
		cmd.Stdout = cmd.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(logWriter.String())
			panic(err)
		}

		urlFriendlyGamename := strings.Replace(path.Base(gitURL), ".git", "", 1)
		urlFriendlyPlatformName := makeUrlFriendly(platformName)
		pckName := fmt.Sprintf("%s-%s.pck", urlFriendlyGamename, urlFriendlyPlatformName)
		targetPath := filepath.Join(os.Getenv("STORAGE_PATH"), "pcks", pckName)
		logrus.WithField("target_path", targetPath).Info("build pck")

		// todo move pck from tmp to persistent storage
		if err := os.Rename("/tmp/foo/export.pck", targetPath); err != nil {
			panic(err)
		}

		pckFiles = append(pckFiles, &PckMetadata{
			Filename:         pckName,
			Platform:         urlFriendlyPlatformName,
			OriginRepository: gitURL,
			Gamename:         gameName,
			MainScene:        mainScene,
		})
	}

	return pckFiles
}

func makeUrlFriendly(input string) string {
	output := &strings.Builder{}
	reader := strings.NewReader(strings.ToLower(input))
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return output.String()
			} else {
				panic(err)
			}
		}

		if r >= 0x30 && r <= 0x39 {
			output.WriteRune(r)
		}
		if r >= 0x61 && r <= 0x7a {
			output.WriteRune(r)
		}
	}
}

func readPresetNames() [][2]string {
	exports := [][2]string{}
	if _, err := os.Stat("/tmp/foo/export_presets.cfg"); err != nil {
		return exports
	}

	exportCfg := ini.Load("/tmp/foo/export_presets.cfg")
	presetCounter := 0
	for {
		exportSec, err := exportCfg.GetSection(fmt.Sprintf("preset.%d", presetCounter))
		presetCounter++
		if err != nil {
			return exports
		}

		presetName, err := exportSec.GetValue("name")
		if err != nil {
			return exports
		}

		presetPlatform, err := exportSec.GetValue("platform")
		if err != nil {
			return exports
		}

		presetName = strings.ReplaceAll(presetName, "\"", "")

		exports = append(exports, [2]string{presetName, presetPlatform})
	}
}

func readGamename() (string, string) {
	if _, err := os.Stat("/tmp/foo/project.godot"); err != nil {
		return "", ""
	}

	godotProjectCfg := ini.Load("/tmp/foo/project.godot")
	name, err := godotProjectCfg.GetValue("application", "config/name")
	if err != nil {
		logrus.WithError(err).Error("unable to get game name")
		return "", ""
	}
	name = strings.ReplaceAll(name, "\"", "")

	initialScene, err := godotProjectCfg.GetValue("application", "run/main_scene")
	if err != nil {
		logrus.WithError(err).Error("unable to get main scene")
		return "", ""
	}
	initialScene = strings.ReplaceAll(initialScene, "\"", "")

	return name, initialScene
}
