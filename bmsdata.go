package gobms

type BmsData struct {
	Path       string
	Title      string
	Subtitle   string
	Playlevel  string
	Difficulty string
	Artist     string
	Genre      string
	Keymode    int // 5, 7, 9, 10, 14, 24, 48
	Md5        string
	Sha256     string
	TotalNotes int

	UniqueBmsData   *UniqueBmsData
	UniqueBmsonData *UniqueBmsonData
}

type UniqueBmsData struct {
	WavDefs map[string]string
	BmpDefs map[string]string
	//WavObjs []
}

func NewUniqueBmsData() *UniqueBmsData {
	var data UniqueBmsData
	data.WavDefs = map[string]string{}
	data.BmpDefs = map[string]string{}
	return &data
}

/*type indexedValue struct {
	Index string
	Value string
}*/

type UniqueBmsonData struct {
}

func NewBmsData() BmsData {
	var bf BmsData
	bf.Keymode = 7
	return bf
}

type BmsDirectory struct {
	Path       string
	Name       string
	BmsDataSet []BmsData
}

func NewBmsDirectory() BmsDirectory {
	var bd BmsDirectory
	bd.BmsDataSet = make([]BmsData, 0)
	return bd
}
