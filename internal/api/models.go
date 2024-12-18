package api

type RequestData struct {
	REDUserID   int     `json:"red_user_id,omitempty"`
	OPSUserID   int     `json:"ops_user_id,omitempty"`
	TorrentID   int     `json:"torrent_id,omitempty"`
	REDKey      string  `json:"red_apikey,omitempty"`
	OPSKey      string  `json:"ops_apikey,omitempty"`
	MinRatio    float64 `json:"minratio,omitempty"`
	RecordLabel string  `json:"record_labels,omitempty"`
	Indexer     string  `json:"indexer"`
}

type ResponseData struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Username string `json:"username"`
		Stats    struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
		Group struct {
			Name      string `json:"name"`
			MusicInfo struct {
				Artists []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"musicInfo"`
		} `json:"group"`
		Torrent *struct {
			Username        string `json:"username"`
			Size            int64  `json:"size"`
			RecordLabel     string `json:"remasterRecordLabel"`
			ReleaseName     string `json:"filePath"`
			CatalogueNumber string `json:"remasterCatalogueNumber"`
		} `json:"torrent"`
	} `json:"response"`
}
