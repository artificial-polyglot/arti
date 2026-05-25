package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/artificial-polyglot/arti/cli_misc"
	"github.com/artificial-polyglot/arti/fetch"
	log "github.com/artificial-polyglot/arti/logger"
	"os"
	"strconv"
)

/*
This go file reads the /download/list endpoint, and accumulates all of the
audio filesets in []TSData
*/

type DownloadList struct {
	TType     string `json:"type"`
	Language  string `json:"language"`
	FilesetId string `json:"fileset_id"`
}
type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}
type Pagination1 struct {
	Page Pagination `json:"pagination"`
}
type DownloadResp struct {
	Data []DownloadList `json:"data"`
	Meta Pagination1    `json:"meta"`
}

func downloadFilestList() []cli_misc.TSData {
	var results []cli_misc.TSData
	ctx := context.Background()
	var page = 0
	for {
		page++
		url := fetch.HOST + `download/list?page=` + strconv.Itoa(page) + `&v=4`
		content, status := fetch.HttpGet(ctx, url, `fetch allowed filesets`)
		catchStatus(status)
		//fmt.Println(string(content))
		var list DownloadResp
		err := json.Unmarshal(content, &list)
		catchErr(err)
		fmt.Println(list.Meta)
		for _, item := range list.Data {
			var rec cli_misc.TSData
			rec.MediaType = item.TType
			rec.MediaId = item.FilesetId
			results = append(results, rec)
		}
		if list.Meta.Page.CurrentPage == list.Meta.Page.TotalPages {
			break
		}
	}
	//var jsonContent []byte
	jsonContent, err := json.MarshalIndent(results, "", "    ")
	catchErr(err)
	err = os.WriteFile(`cli_misc/find_timestamps/FilesetList.json`, jsonContent, 0644)
	catchErr(err)
	return results
}

func catchErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func catchStatus(status *log.Status) {
	if status != nil {
		fmt.Println(status)
		os.Exit(1)
	}
}
