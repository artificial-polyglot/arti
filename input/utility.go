package input

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func Glob(ctx context.Context, search string) ([]InputFile, *log.Status) {
	var results []InputFile
	if search != `` {
		files, err := filepath.Glob(search)
		if err != nil {
			return results, log.Error(ctx, 500, err, "Error expanding file glob pattern:", search)
		}
		for _, file := range files {
			var input InputFile
			input.Directory = filepath.Dir(file)
			input.Filename = filepath.Base(file)
			results = append(results, input)
		}
	}
	return results, nil
}

func FillInputFile(ctx context.Context, testament request.Testament, files []InputFile) ([]InputFile, *log.Status) {
	var status *log.Status
	files, status = unzip(ctx, files)
	if status != nil {
		return files, status
	}
	for i := range files {
		if files[i].MediaType == "" {
			status = setMediaType(ctx, &files[i])
			if status != nil {
				return files, status
			}
		}
		status = parseFilenames(ctx, &files[i])
		if status != nil {
			return files, status
		}
	}
	files = pruneBooksByRequest(files, testament)
	return files, nil
}

func unzip(ctx context.Context, files []InputFile) ([]InputFile, *log.Status) {
	var results []InputFile
	if len(files) == 1 && filepath.Ext(files[0].Filename) == `.zip` {
		r, err := zip.OpenReader(files[0].FilePath())
		if err != nil {
			return results, log.Error(ctx, 500, err, "Error opening zip file:", files[0].FilePath())
		}
		defer r.Close()
		dest := files[0].Directory
		for _, f := range r.File {
			if f.FileInfo().Name()[0] == '.' {
				continue
			}
			rc, err2 := f.Open()
			if err2 != nil {
				return results, log.Error(ctx, 500, err2, "Error opening entry inside zip:", f.FileInfo().Name())
			}
			defer rc.Close()
			if f.FileInfo().IsDir() {
				dest = filepath.Join(dest, f.FileInfo().Name())
				err = os.MkdirAll(dest, f.Mode())
				if err != nil {
					return results, log.Error(ctx, 500, err, "Error creating directory during zip extraction:", dest)
				}
				continue
			}
			path := filepath.Join(dest, f.FileInfo().Name())
			outFile, err3 := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err3 != nil {
				return results, log.Error(ctx, 500, err3, "Error creating file during zip extraction:", path)
			}
			defer outFile.Close()
			_, err = io.Copy(outFile, rc)
			_ = rc.Close()
			if err != nil {
				return results, log.Error(ctx, 500, err, "Error extracting file from zip:", f.FileInfo().Name())
			}
			results = append(results, InputFile{Filename: f.FileInfo().Name(), Directory: dest})
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Filename < results[j].Filename
		})
	} else {
		results = files
	}
	return results, nil
}

// setMediaType function looks at names and sets the Media Type
func setMediaType(ctx context.Context, file *InputFile) *log.Status {
	fN := file.Filename
	fNLower := strings.ToLower(fN)
	if strings.HasSuffix(fN, `_ET`) || strings.HasSuffix(fN, `_ET.json`) {
		file.MediaType = request.TextPlainEdit
	} else if strings.HasSuffix(fNLower, `usx`) {
		file.MediaType = request.TextUSXEdit
	} else if strings.HasSuffix(fNLower, `.sfm`) || strings.HasSuffix(fNLower, `.usfm`) {
		file.MediaType = request.TextUSFMEdit
	} else if (fN[0] == 'A' || fN[0] == 'B') && (fN[1] >= '0' && fN[1] <= '9') {
		file.MediaType = request.Audio
	} else if strings.HasSuffix(fNLower, `.xlsx`) || strings.HasSuffix(fNLower, `.xlsm`) {
		file.MediaType = request.TextScript
	} else if strings.HasSuffix(fNLower, `.csv`) {
		file.MediaType = request.TextCSV
	} else if (fN[0] == 'N' || fN[0] == 'O' || fN[0] == 'P') && fN[1] == '1' && fN[2] == '_' {
		file.MediaType = request.Audio
	} else if (fN[0] == 'N' || fN[0] == 'O' || fN[0] == 'P') && fN[1] == '2' && fN[2] == '_' {
		file.MediaType = request.AudioDrama
	} else if strings.HasSuffix(fNLower, `.mp3`) || strings.HasSuffix(fNLower, `.wav`) {
		file.MediaType = request.Audio
	} else {
		parts := strings.Split(fN, `_`)
		if len(parts) > 2 {
			ord := parts[1]
			if (ord[0] == 'A' || ord[0] == 'B') && (ord[1] >= '0' && ord[1] <= '9') {
				file.MediaType = request.Audio
			}
		}
	}
	if file.MediaType == `` {
		return log.ErrorNoErr(ctx, 400, `Could not identify media type from filename for:`, file.Filename)
	}
	return nil
}

func parseFilenames(ctx context.Context, file *InputFile) *log.Status {
	var status *log.Status
	if file.MediaType == request.TextPlain || file.MediaType == request.TextPlainEdit ||
		file.MediaType == request.TextCSV {
		file.MediaId = strings.Split(file.Filename, `.`)[0]
		test := file.Filename[6]
		if test == 'O' {
			file.Testament = `OT`
		} else if test == 'N' {
			file.Testament = `NT`
		}
		file.FileExt = filepath.Ext(file.Filename)
	} else if file.MediaType == request.TextUSXEdit || file.MediaType == request.TextUSFMEdit {
		parts := strings.Split(file.Directory, `/`)
		file.MediaId = parts[len(parts)-1]
		file.BookId = findTextBookId(file.Filename)
		if file.BookId == "" {
			return log.ErrorNoErr(ctx, 400, "Unable to find bookId in", file.Filename)
		}
		seq, found := db.BookSeqMap[file.BookId]
		if found {
			file.BookSeq = strconv.Itoa(seq)
		}
		file.Testament = db.Testament(file.BookId)
		file.FileExt = filepath.Ext(file.Filename)
	} else if file.MediaType == request.TextScript {
		file.MediaId = strings.Split(file.Filename, `.`)[0]
		test := file.Filename[6]
		if test == 'O' {
			file.Testament = `OT`
		} else if test == 'N' {
			file.Testament = `NT`
		}
		file.FileExt = filepath.Ext(file.Filename)
	} else if file.MediaType == request.Audio || file.MediaType == request.AudioDrama {
		fN := file.Filename
		if strings.HasSuffix(fN, `VOX.mp3`) || strings.HasSuffix(fN, `VOX.wav`) {
			parts := strings.Split(fN, `_`)
			if len(parts[len(parts)-2]) >= 5 {
				file.ScriptLine = parts[len(parts)-2] // Filled later using db.conn
			} else {
				status = parseVOXAudioFilename(ctx, file)
			}
		} else if (fN[0] == 'A' || fN[0] == 'B') && (fN[1] >= '0' && fN[1] <= '9') {
			status = parseV2AudioFilename(ctx, file)
		} else {
			status = parseV4AudioFilename(ctx, file)
		}
		if status != nil {
			return status
		}
	} else {
		status = log.ErrorNoErr(ctx, 400, `Type must be one of "text_plain", "text_plain_edit", "text_usx", "audio"`)
	}
	return status
}

func parseV2AudioFilename(ctx context.Context, file *InputFile) *log.Status {
	var status *log.Status
	var err error
	file.FileExt = filepath.Ext(file.Filename)
	filename := file.Filename[:len(file.Filename)-len(file.FileExt)]
	ab := filename[0]
	if ab == 'A' {
		file.Testament = `OT`
	} else if ab == 'B' {
		file.Testament = `NT`
	}
	seq := filename[1:4]
	file.BookSeq = strings.Trim(seq, `_`)
	chapter := strings.Trim(filename[5:8], `_`)
	file.Chapter, err = strconv.Atoi(chapter)
	if err != nil {
		return log.Error(ctx, 500, err, "Error converting chapter to int in audio filename:", file.Filename,
			"chapter part:", chapter)
	}
	book := strings.Trim(filename[9:21], `_`)
	file.BookId = db.USFMBookId(ctx, book)
	file.MediaId = filename[21:]
	return status
}

func parseV4AudioFilename(ctx context.Context, file *InputFile) *log.Status {
	var status *log.Status
	var err error
	file.FileExt = filepath.Ext(file.Filename)
	filename := file.Filename[:len(file.Filename)-len(file.FileExt)]
	filename = strings.Replace(filename, `-`, `_`, -1)
	parts := strings.Split(filename, `_`)
	file.MediaId = parts[0]
	if len(parts) > 1 {
		ab := parts[1][0]
		if ab == 'A' {
			file.Testament = `OT`
		} else if ab == 'B' {
			file.Testament = `NT`
		}
		file.BookSeq = parts[1][1:]
	}
	if len(parts) > 2 {
		file.BookId, status = validateBookId(ctx, parts[2])
		if status != nil {
			return status
		}
	}
	if len(parts) > 3 {
		file.Chapter, err = strconv.Atoi(parts[3])
		if err != nil {
			return log.Error(ctx, 500, err, `Error convert chapter to int`, parts[3])
		}
	}
	if len(parts) > 4 {
		file.Verse = parts[4]
	}
	if len(parts) > 5 {
		file.ChapterEnd, err = strconv.Atoi(parts[5])
		if err != nil {
			return log.Error(ctx, 500, err, `Error convert chapterEnd to int`, parts[5])
		}
	}
	if len(parts) > 6 {
		file.VerseEnd = parts[6]
	}
	return status
}

func parseVOXAudioFilename(ctx context.Context, file *InputFile) *log.Status {
	var status *log.Status
	var err error
	parts := strings.Split(file.Filename, `_`)
	if len(parts) != 7 {
		return log.ErrorNoErr(ctx, 500, "A VOX. filename is expected to have 7 parts", parts)
	}
	drama := parts[0]
	langCode := parts[1]
	versionCode := parts[2]
	file.BookSeq = parts[3]
	file.BookId, file.Chapter, err = findAudioBookId(parts)
	if err != nil {
		return log.Error(ctx, 500, err, "Unable to find book code or chapter in ", file.Filename)
	}
	if parts[0][0] == 'N' {
		file.Testament = `NT`
	} else if parts[0][0] == 'O' {
		file.Testament = `OT`
	} else if parts[0][0] == 'P' {
		file.Testament = db.Testament(file.BookId)
	} else {
		return log.ErrorNoErr(ctx, 500, "Unknown media type", parts[0])
	}
	file.MediaId = langCode + versionCode + drama + "DA"
	return status
}

func pruneBooksByRequest(files []InputFile, testament request.Testament) []InputFile {
	var results []InputFile
	for _, f := range files {
		if testament.Has(f.Testament, f.BookId) || f.BookId == `` {
			results = append(results, f)
		}
	}
	return results
}

func UpdateIdent(conn db.DBAdapter, ident *db.Ident, textFiles []InputFile, audioFiles []InputFile) *log.Status {
	var status *log.Status
	_ = updateIdentText(ident, textFiles)
	_ = updateIdentAudio(ident, audioFiles)
	status = conn.InsertReplaceIdent(*ident)
	return status
}

func updateIdentText(ident *db.Ident, files []InputFile) bool {
	var result = false
	if len(files) > 0 {
		f := files[0]
		if f.MediaType != request.TextNone {
			ident.TextSource = f.MediaType
		}
		result = true
		if f.Testament == `OT` {
			ident.TextOTId = f.MediaId
		} else if f.Testament == `NT` {
			ident.TextNTId = f.MediaId
		}
	}
	return result
}

func updateIdentAudio(ident *db.Ident, files []InputFile) bool {
	var result = false
	for _, f := range files {
		result = true
		if f.Testament == `OT` {
			ident.AudioOTId = f.MediaId
		} else if f.Testament == `NT` {
			ident.AudioNTId = f.MediaId
		}
	}
	return result
}

/*
When this function finds a book code that starts with a numeric character,
it checks the next three chars to see if that looks like a book code also.
This is because 1TI and TIT are both valid book codes.
*/
func findTextBookId(filename string) string {
	dot := strings.Index(filename, ".")
	for i := 0; i <= dot-3; i++ {
		code := filename[i : i+3]
		bookId := isBookId(code)
		if bookId != "" {
			if bookId[0] >= 'A' || i+4 > dot {
				return bookId
			} else {
				code = filename[i+1 : i+4]
				bookId2 := isBookId(code)
				if bookId2 != "" {
					return bookId2
				} else {
					return bookId
				}
			}
		}
	}
	return ""
}

/*
This method searches an underscore delimited filename backwards.
When it finds the bookId, it assumes the following one is the chapter
*/
func findAudioBookId(parts []string) (string, int, error) {
	//baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	//parts := strings.Split(baseName, "_")
	for i := len(parts) - 2; i >= 0; i-- {
		code := parts[i]
		bookId := isBookId(code)
		if bookId != "" {
			chapter, err := strconv.Atoi(parts[i+1])
			return bookId, chapter, err
		}
	}
	return "", 0, errors.New("No valid book code on filename " + strings.Join(parts, "_"))
}

func isBookId(code string) string { //(string, int, int) {
	corrected, found := corrections[code]
	if found {
		code = corrected
	}
	_, ok := db.BookChapterMap[code]
	if ok {
		return code
	} else {
		return ""
	}
}

var corrections = map[string]string{
	"PSM": "PSA", // Psalms
	"PRV": "PRO", // Proverbs
	"SOS": "SNG", // Song of Solomon
	"EZE": "EZK", // Ezekiel
	"JOE": "JOL", // Joel
	"NAH": "NAM", // Nahum
	"MRC": "MRK", // Mark
	"LUC": "LUK", // Luke
	"JUA": "JHN", // John
	"HEC": "ACT", // Acts
	"EFE": "EPH", // Ephesians
	"FHP": "PHP", // Philippians
	"1TE": "1TH", // 1 Thessalonians
	"2TE": "2TH", // 2 Thessalonians
	"TTO": "TIT", // Titus
	"TTL": "TIT", // Titus
	"TTS": "TIT", // Titus
	"FHM": "PHM", // Philemon
	"JMS": "JAS", // James
	"SNT": "JAS", // James
	"APO": "REV", // Revelation
}

func validateBookId(ctx context.Context, bookId string) (string, *log.Status) {
	corrected, found := corrections[bookId]
	if found {
		bookId = corrected
	}
	_, ok := db.BookChapterMap[bookId]
	if !ok {
		return bookId, log.ErrorNoErr(ctx, 500, "Unrecognised book ID:", bookId)
	}
	return bookId, nil
}

// Parse DBP4 Audio names
//{mediaid}_{A/B}{ordering}_{USFM book code}_{chapter start}[_{verse start}-{chapter stop}_{verse stop}].mp3|webm
//eg: ENGESVN2DA_B001_MAT_001.mp3  (full chapter)
//eg: IRUNLCP1DA_B013_1TH_001_001-001_010.mp3  (partial chapter, verses 1-10)
