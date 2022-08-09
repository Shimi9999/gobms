package gobms

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			return bmsData, err
		}
		return bmsData, nil
	}
	return _loadBms(path)
}

func _loadBms(path string) (bmsData BmsData, _ error) {
	file, err := os.Open(path)
	if err != nil {
		return bmsData, fmt.Errorf("bmsData open error: %w", err)
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
	chmap := map[string]bool{"7k": false, "10k": false, "14k": false}
	for scanner.Scan() {
		line, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), scanner.Text())
		if err != nil {
			return bmsData, fmt.Errorf("ShiftJIS decode error: %w", err)
		}

		// TODO 小文字に対応 (#title,#difficultyなど)
		if strings.HasPrefix(line, "#TITLE") {
			bmsData.Title = strings.Trim(line[6:], " ")
		} else if strings.HasPrefix(line, "#SUBTITLE") {
			bmsData.Subtitle = strings.Trim(line[9:], " ")
		} else if strings.HasPrefix(line, "#PLAYLEVEL") {
			bmsData.Playlevel = strings.Trim(line[10:], " ")
		} else if strings.HasPrefix(line, "#DIFFICULTY") {
			bmsData.Difficulty = strings.Trim(line[11:], " ")
		} else if strings.HasPrefix(line, "#ARTIST") {
			bmsData.Artist = strings.Trim(line[7:], " ")
		} else if strings.HasPrefix(line, "#GENRE") {
			bmsData.Genre = strings.Trim(line[6:], " ")
		} else if regexp.MustCompile(`#[0-9]{5}:.+`).MatchString(line) {
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
		return bmsData, fmt.Errorf("Get bmshash error: %w", err)
	}

	return bmsData, nil
}

func LoadBmson(path string) (bmsData BmsData, _ error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return bmsData, fmt.Errorf("BMSONfile open error: %w", err)
	}

	var bmson Bmson
	if err := json.Unmarshal(bytes, &bmson); err != nil {
		return bmsData, fmt.Errorf("BMSONfile unmarshal error: %w", err)
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
		return bmsData, fmt.Errorf("Get bmshash error: %w", err)
	}

	bmsData.TotalNotes = 1 // TODO ちゃんとトータルノーツ数える

	return bmsData, nil
}

func getFileHash(path string) (string, string, error) {
	bmsStr, err := loadFileString(path)
	if err != nil {
		return "", "", err
	}
	md5 := fmt.Sprintf("%x", md5.Sum([]byte(bmsStr)))
	sha256 := fmt.Sprintf("%x", sha256.Sum256([]byte(bmsStr)))

	return md5, sha256, nil
}

func loadFileString(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("File open error: %w", err)
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
			return "", fmt.Errorf("File read error: %w", err)
		}

		str += string(buf[:n])
	}
	return str, nil
}

func LoadBmsInDirectory(path string) (BmsDirectory, error) {
	bmsdirectory := NewBmsDirectory()
	bmsdirectory.Path = path
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		if IsBmsPath(f.Name()) {
			var bmsData BmsData
			var err error
			bmspath := filepath.Join(path, f.Name())
			bmsData, err = LoadBms(bmspath)
			if err != nil {
				return NewBmsDirectory(), err
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
	files, _ := ioutil.ReadDir(path)
	bmsExist := false
	for _, f := range files {
		if IsBmsPath(f.Name()) {
			bmsdirectory, err := LoadBmsInDirectory(path)
			if err != nil {
				return err
			}
			*bmsdirs = append(*bmsdirs, bmsdirectory)
			bmsExist = true
			break
		}
	}
	if !bmsExist {
		for _, f := range files {
			if f.IsDir() {
				err := FindBmsInDirectory(filepath.Join(path, f.Name()), bmsdirs)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
