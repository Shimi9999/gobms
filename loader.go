package gobms

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func LoadBms(path string) (bmsData BmsData, err error) {
	if IsBmsonPath(path) {
		bmsData, err = LoadBmson(path)
		if err != nil {
			return bmsData, fmt.Errorf("LoadBmson: %w", err)
		}
		return bmsData, nil
	}
	return _loadBms(path)
}

var COMMANDS = []string{"title", "subtitle", "playlevel", "difficulty", "artist", "genre"}
var INDEXED_COMMANDS = []string{"wav", "bmp" /*, "bpm", "stop", "scroll"*/}

func _loadBms(path string) (bmsData BmsData, _ error) {
	file, err := os.Open(path)
	if err != nil {
		return bmsData, fmt.Errorf("bmsData Open: %w", err)
	}
	defer file.Close()

	const (
		initialBufSize = 10000
		maxBufSize     = 1000000
	)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, initialBufSize)
	scanner.Buffer(buf, maxBufSize)

	bmsData = NewBmsData()
	bmsData.Path = path
	bmsData.UniqueBmsData = NewUniqueBmsData()
	chmap := map[string]bool{"7k": false, "10k": false, "14k": false}
	for scanner.Scan() {
		line, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), scanner.Text())
		if err != nil {
			return bmsData, fmt.Errorf("ShiftJIS decode: %w", err)
		}

		for _, command := range COMMANDS {
			if strings.HasPrefix(strings.ToLower(line), "#"+command) {
				value := strings.TrimSpace(line[len(command)+1:])

				switch command {
				case "title":
					bmsData.Title = value
				case "subtitle":
					bmsData.Subtitle = value
				case "playlevel":
					bmsData.Playlevel = value
				case "difficulty":
					bmsData.Difficulty = value
				case "artist":
					bmsData.Artist = value
				case "genre":
					bmsData.Genre = value
				}
				goto lineReadComplete
			}
		}

		for _, command := range INDEXED_COMMANDS {
			if regexp.MustCompile(`#` + command + `[0-9a-z]{2} .+`).MatchString(strings.ToLower(line)) {
				index := line[1+len(command) : 1+len(command)+2]
				value := strings.TrimSpace(line[1+len(command)+3:])

				switch command {
				case "wav":
					bmsData.UniqueBmsData.WavDefs[index] = value
				case "bmp":
					bmsData.UniqueBmsData.BmpDefs[index] = value
				}
				goto lineReadComplete
			}
		}

		// TODO オブジェも読む？
		if regexp.MustCompile(`#[0-9]{3}[0-9a-z]{2}:.+`).MatchString(line) {
			chint, _ := strconv.Atoi(line[4:6])
			if (chint >= 18 && chint <= 19) || (chint >= 38 && chint <= 39) {
				chmap["7k"] = true
			} else if (chint >= 21 && chint <= 26) || (chint >= 41 && chint <= 46) {
				chmap["10k"] = true
			} else if (chint >= 28 && chint <= 29) || (chint >= 48 && chint <= 49) {
				chmap["14k"] = true
			}

			if (chint >= 11 && chint <= 19) || (chint >= 21 && chint <= 29) {
				bmsData.TotalNotes++
				// TODO キーモード毎に検出範囲を変える
			}
		}

	lineReadComplete:
	}
	if scanner.Err() != nil {
		return bmsData, fmt.Errorf("bmsData scan error: %w", scanner.Err())
	}

	if filepath.Ext(path) == ".pms" {
		bmsData.Keymode = 9
	} else if chmap["10k"] || chmap["14k"] {
		if chmap["7k"] || chmap["14k"] {
			bmsData.Keymode = 14
		} else {
			bmsData.Keymode = 10
		}
	} else if chmap["7k"] {
		bmsData.Keymode = 7
	} else {
		bmsData.Keymode = 5
	}

	bmsData.Md5, bmsData.Sha256, err = getFileHash(path)
	if err != nil {
		return bmsData, fmt.Errorf("getFileHash: %w", err)
	}

	return bmsData, nil
}

func LoadBmson(path string) (bmsData BmsData, _ error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return bmsData, fmt.Errorf("ReadFile: %w", err)
	}

	var bmson Bmson
	if err := json.Unmarshal(bytes, &bmson); err != nil {
		return bmsData, fmt.Errorf("Unmarshal: %w", err)
	}

	bmsData = NewBmsData()
	bmsData.Path = path
	bmsData.Title = bmson.Bmsoninfo.Title + bmson.Bmsoninfo.Subtitle
	bmsData.Subtitle = bmson.Bmsoninfo.Chartname
	bmsData.Playlevel = strconv.Itoa(bmson.Bmsoninfo.Level)
	bmsData.Artist = bmson.Bmsoninfo.Artist
	bmsData.Genre = bmson.Bmsoninfo.Genre

	keymap := map[string]int{"5k": 5, "7k": 7, "9k": 9, "10k": 10, "14k": 14, "keyboard-24k-double": 48, "24k": 24}
	for key, value := range keymap {
		if strings.Contains(bmson.Bmsoninfo.Modehint, key) {
			bmsData.Keymode = value
			break
		}
	}

	_, bmsData.Sha256, err = getFileHash(path)
	if err != nil {
		return bmsData, fmt.Errorf("getFileHash: %w", err)
	}

	bmsData.TotalNotes = 1 // TODO ちゃんとトータルノーツ数える

	return bmsData, nil
}

func getFileHash(path string) (string, string, error) {
	bmsStr, err := loadFileString(path)
	if err != nil {
		return "", "", fmt.Errorf("loadFileString (%s): %w", path, err)
	}
	md5 := fmt.Sprintf("%x", md5.Sum([]byte(bmsStr)))
	sha256 := fmt.Sprintf("%x", sha256.Sum256([]byte(bmsStr)))

	return md5, sha256, nil
}

func loadFileString(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("File Open: %w", err)
	}
	defer file.Close()

	var str string
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			return "", fmt.Errorf("File Read: %w", err)
		}

		str += string(buf[:n])
	}
	return str, nil
}

func LoadBmsInDirectory(path string) (BmsDirectory, error) {
	bmsdirectory := NewBmsDirectory()
	bmsdirectory.Path = path
	files, _ := os.ReadDir(path)
	for _, f := range files {
		if IsBmsPath(f.Name()) {
			var bmsData BmsData
			var err error
			bmspath := filepath.Join(path, f.Name())
			bmsData, err = LoadBms(bmspath)
			if err != nil {
				return NewBmsDirectory(), fmt.Errorf("LoadBms (%s): %w", bmspath, err)
			}
			if bmsData.TotalNotes > 0 {
				bmsdirectory.BmsDataSet = append(bmsdirectory.BmsDataSet, bmsData)
			}
		}
	}
	if len(bmsdirectory.BmsDataSet) > 0 {
		bmsdirectory.Name = bmsdirectory.BmsDataSet[0].Title
	}

	return bmsdirectory, nil
}

func FindBmsInDirectory(path string, bmsdirs *[]BmsDirectory) error {
	files, _ := os.ReadDir(path)
	bmsExist := false
	for _, f := range files {
		if IsBmsPath(f.Name()) {
			bmsdirectory, err := LoadBmsInDirectory(path)
			if err != nil {
				return fmt.Errorf("LoadBmsInDirectory (%s): %w", path, err)
			}
			*bmsdirs = append(*bmsdirs, bmsdirectory)
			bmsExist = true
			break
		}
	}
	if !bmsExist {
		for _, f := range files {
			if f.IsDir() {
				dirPath := filepath.Join(path, f.Name())
				err := FindBmsInDirectory(dirPath, bmsdirs)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
