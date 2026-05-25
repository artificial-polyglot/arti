package fetch

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/decode_yaml/request"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlainText(t *testing.T) {
	ctx := context.Background()
	var req request.Request
	req.AudioData.NoAudio = true
	req.TextData.BibleBrain.TextPlain = true
	req.Testament.OT = true
	req.Testament.NT = true
	client := NewAPIDBPClient(ctx, `ENGWEB`)
	info, status := client.BibleInfo()
	if status != nil {
		t.Error(`BibleInfo Error`, status.Err)
	}
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	fmt.Println("INFO", info)
	//ident := client.CreateIdent(info)
	//fmt.Println("IDENT", ident)
	if info.AudioOTFileset.Id != `` || info.AudioNTFileset.Id != `` {
		t.Error(`Should have found no audio filesets.`)
	}
	if info.TextNTUSXFileset.Id == `` || info.TextOTUSXFileset.Id == `` ||
		info.TextNTPlainFileset.Id == `` || info.TextOTPlainFileset.Id == `` {
		t.Error(`Should have found two text filesets`)
	}
	err := deleteFile(info.BibleId, info.TextOTPlainFileset)
	if err != nil {
		t.Error(`Did not delete file.`)
	}
	download := NewAPIDownloadClient(ctx, info.BibleId, req.Testament)
	status = download.Download(info)
	if status != nil {
		t.Error("Unexpected Error", status.Message)
	}
	count, err := countFiles(info.BibleId, info.TextOTPlainFileset)
	if err != nil {
		t.Error(`Download err was not expected`, err)
	}
	if count != 2 {
		t.Error("Two text files are expected")
	}
}

func TestUSXDownload(t *testing.T) {
	ctx := context.Background()
	var req request.Request
	req.AudioData.NoAudio = true
	req.TextData.BibleBrain.TextUSXEdit = true
	req.Testament.NT = true
	client := NewAPIDBPClient(ctx, `ENGWEB`)
	info, status := client.BibleInfo()
	if status != nil {
		t.Error(`BibleInfo Error`, status.Err)
	}
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	//_ = client.CreateIdent(info)
	if info.AudioOTFileset.Id != `` || info.AudioNTFileset.Id != `` {
		t.Error(`Should have found no audio filesets.`)
	}
	if info.TextNTUSXFileset.Id == `` || info.TextOTUSXFileset.Id == `` ||
		info.TextNTPlainFileset.Id == `` || info.TextOTPlainFileset.Id == `` {
		t.Error(`Should have found one text fileset`)
	}
	err := deleteFile(info.BibleId, info.TextNTUSXFileset)
	if err != nil {
		t.Error(`Did not delete file.`)
	}
	download := NewAPIDownloadClient(ctx, info.BibleId, req.Testament)
	status = download.Download(info)
	if status != nil {
		t.Error(`Download Err is unexpected`, status.Message)
	}
	count, err := countFiles(info.BibleId, info.TextNTUSXFileset)
	if count != 27 {
		t.Error("27 books in NT are expected, found:", count)
	}
	if err != nil {
		t.Error(`Download err was not expected`, err)
	}
}

func TestAudioDownload(t *testing.T) {
	ctx := context.Background()
	var req request.Request
	req.AudioData.BibleBrain.MP3_64 = true
	req.TextData.NoText = true
	req.Testament.NT = true
	client := NewAPIDBPClient(ctx, `ENGWEB`)
	info, status := client.BibleInfo()
	if status != nil {
		t.Error(`BibleInfo Error`, status.Err)
	}
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	fmt.Println("INFO", info)
	//_ = client.CreateIdent(info)
	if info.AudioOTFileset.Id != `` || info.AudioNTFileset.Id == `` {
		t.Error(`Should have found no audio filesets.`)
	}
	if info.TextNTUSXFileset.Id == `` || info.TextOTUSXFileset.Id == `` ||
		info.TextNTPlainFileset.Id == `` || info.TextOTPlainFileset.Id == `` {
		t.Error(`Should have found no text fileset`)
	}
	err := deleteFile(info.BibleId, info.AudioNTFileset)
	if err != nil {
		t.Error(`Did not delete file.`)
	}
	download := NewAPIDownloadClient(ctx, info.BibleId, req.Testament)
	status = download.Download(info)
	if status != nil {
		t.Error(`Download Err is unexpected`, status.Message)
	}
	count, err := countFiles(info.BibleId, info.AudioNTFileset)
	if count != 260 {
		t.Error("260 chapters in NT are expected, found:", count)
	}
	if err != nil {
		t.Error(`Download err was not expected`, err)
	}
}

func Test403AndDownload(t *testing.T) {
	ctx := context.Background()
	var req request.Request
	req.AudioData.BibleBrain.MP3_64 = true
	req.TextData.NoText = true
	req.Testament.NT = true
	client := NewAPIDBPClient(ctx, `ENGESV`)
	info, status := client.BibleInfo()
	if status != nil {
		t.Error(`BibleInfo Error`, status.Err)
	}
	client.FindFilesets(&info, req.AudioData.BibleBrain, req.TextData.BibleBrain, req.Testament)
	//_ = client.CreateIdent(info)
	if info.AudioOTFileset.Id != `` || info.AudioNTFileset.Id == `` {
		t.Error(`Should have found one audio fileset.`)
	}
	if info.TextNTUSXFileset.Id == `` || info.TextOTUSXFileset.Id == `` ||
		info.TextNTPlainFileset.Id == `` || info.TextOTPlainFileset.Id == `` {
		t.Error(`Should have found no text fileset`)
	}
	err := deleteFile(info.BibleId, info.AudioNTFileset)
	if err != nil {
		t.Error(`Did not delete file.`)
	}
	download := NewAPIDownloadClient(ctx, info.BibleId, req.Testament)
	status = download.Download(info)
	if status != nil {
		t.Error(`Download error is unexpected`, status)
	}
}

func deleteFile(bibleId string, fs FilesetType) error {
	if fs.Type == `text_plain` {
		filePath := filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, fs.Id+`.json`)
		fmt.Println(`Delete`, filePath)
		err := os.Remove(filePath)
		return err
	} else {
		filePath := filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, fs.Id)
		files, err := os.ReadDir(filePath)
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		for _, num := range []int{11, 7, 5, 3, 1} {
			if len(files) > num {
				delPath := filepath.Join(filePath, files[num].Name())
				fmt.Println("Delete", delPath)
				err := os.Remove(delPath)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func countFiles(bibleId string, fs FilesetType) (int, error) {
	var filePath string
	if fs.Type == `text_plain` {
		filePath = filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, `*.json`)
	} else if fs.Type == `text_usx` {
		filePath = filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, fs.Id, `*.usx`)
	} else {
		fileExt := strings.ToLower(fs.Codec)
		filePath = filepath.Join(os.Getenv(`FCBH_DATASET_FILES`), bibleId, fs.Id, `*.`+fileExt)
	}
	files, err := filepath.Glob(filePath)
	return len(files), err
}
