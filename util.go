package gobms

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

func IsBmsPath(path string) bool {
	ext := filepath.Ext(path)
	bmsExts := []string{".bms", ".bme", ".bml", ".pms", ".bmson"}
	for _, be := range bmsExts {
		if strings.ToLower(ext) == be {
			return true
		}
	}
	return false
}

func IsBmsonPath(path string) bool {
	return filepath.Ext(path) == ".bmson"
}

func GetDifficultyFromTitle(bms BmsData) string {
	difficulties := []string{"beginner", "normal", "hyper", "another", "insane"}
	difnums := []string{"1", "2", "3", "4", "5"}
	brackets := [][]string{{`\[`, `\]`}, {`\(`, `\)`}, {"-", "-"}, {`【`, `】`}}

	fulltitle := strings.ToLower(strings.TrimSpace(bms.Title + bms.Subtitle))
	// first match black another(=insane)
	for _, bracket := range brackets {
		s := ".+" + bracket[0] + ".*black.*another.*" + bracket[1] + "$"
		if regexp.MustCompile(s).MatchString(fulltitle) {
			return difnums[4]
		}
	}

	for index, difficulty := range difficulties {
		for _, bracket := range brackets {
			s := ".+" + bracket[0] + ".*" + difficulty + ".*" + bracket[1] + "$"
			if regexp.MustCompile(s).MatchString(fulltitle) {
				return difnums[index]
			}
		}
	}
	return ""
}

func GetDifficultyFromPureName(purename string, justmatch bool) string {
	if purename == "" {
		return ""
	}
	difficulties := []string{}
	predifs := []string{"", "sp", "dp", "5", "7", "9", "14", "5k", "7k", "9k", "14k"}
	difs := []string{"b", "n", "h", "a", "i", "beginner", "normal", "hyper", "another", "insane"}
	for _, predif := range predifs {
		for _, dif := range difs {
			difficulties = append(difficulties, predif+dif)
		}
	}
	pres := []string{"", " ", "-", "_"}
	if justmatch {
		pres = []string{""}
	}
	brackets := [][]string{{`\[`, `\]`}, {`\(`, `\)`}}
	difnums := []string{"1", "2", "3", "4", "5"}

	purename = strings.ToLower(purename)
	for index, difficulty := range difficulties {
		for _, pre := range pres {
			if pre == "" {
				if purename == difficulty {
					return difnums[index%5]
				}
			} else if strings.HasSuffix(purename, pre+difficulty) {
				return difnums[index%5]
			}
		}
		for _, bracket := range brackets {
			s := ".+" + bracket[0] + difficulty + bracket[1] + "$"
			if regexp.MustCompile(s).MatchString(purename) {
				return difnums[index%5]
			}
		}
	}
	return ""
}

func GetDifficultyFromPath(bms BmsData) string {
	return GetDifficultyFromPureName(getPureFileName(bms.Path), false)
}

// 複数のbmsファイル名から先頭一致した文字列を排除してDIFFICULTYを抽出する
// ["bmsN.bms", "bmsH.bms", "BmsA.bms"] → ["N", "H", "A"] → ["2", "3", "4"]
func FindDifficultyFromDirectory(bmsdir *BmsDirectory) {
	if len(bmsdir.BmsDataSet) < 2 {
		return
	}

	purenames := []string{}
	for _, bmsData := range bmsdir.BmsDataSet {
		purenames = append(purenames, strings.ToLower(getPureFileName(bmsData.Path)))
	}

	var matchstr string
	i := 1
	for ; i <= utf8.RuneCountInString(purenames[0]); i++ {
		for j := 1; j < len(purenames); j++ {
			if utf8.RuneCountInString(purenames[j]) < i || purenames[0][:i] != purenames[j][:i] {
				if i == 1 {
					return
				} else {
					goto OUT
				}
			}
		}
	}
OUT:
	matchstr = purenames[0][:i-1]
	for index := range bmsdir.BmsDataSet {
		bmsdir.BmsDataSet[index].Difficulty = GetDifficultyFromPureName(purenames[index][utf8.RuneCountInString(matchstr):], true)
	}
}

func getPureFileName(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

func RemoveSuffixChartName(title string) string {
	title = trimBothSpace(title)
	brackets := [][]string{{`\[`, `\]`}, {`［`, `］`}, {`\(`, `\)`}, {`（`, `）`}, {"-", "-"}, {`【`, `】`}, {"<", ">"}, {"〈", "〉"}, {"⟨", "⟩"}}

	for _, bracket := range brackets {
		r := regexp.MustCompile(".+(" + bracket[0] + "[^" + bracket[0] + "]+" + bracket[1] + ")$")
		//r := regexp.MustCompile(".+(" + bracket[0] + "[^" + bracket[0] + "(?!bms ?edit)]+" + bracket[1] + ")$")
		strs := r.FindStringSubmatch(title)
		indexes := r.FindStringSubmatchIndex(title)
		if len(strs) >= 2 && len(indexes) >= 4 {
			if regexp.MustCompile("bms ?edit").MatchString(strings.ToLower(strs[1])) {
				return title
			}
			return trimBothSpace(title[:indexes[2]])
		}
	}
	return title
}

func trimBothSpace(str string) string {
	return strings.Trim(strings.TrimSpace(str), "　")
}
