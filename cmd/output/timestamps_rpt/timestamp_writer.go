package timestamps_rpt

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func TimestampWriters(ctx context.Context, verses []proofing_rpt.Verse2) []db.Output {
	var reports []db.Output
	dir, err := os.MkdirTemp("", "arti-rpt-*")
	if err != nil {
		log.Warn(ctx, err, "Could not Marshal Timestamps")
		return reports
	}
	jsonOut := JSONTimestampHierachyRpt(ctx, dir, verses)
	reports = append(reports, jsonOut)
	scriptsOut := CSVScripts(ctx, dir, verses)
	reports = append(reports, scriptsOut)
	wordsOut := CSVWords(ctx, dir, verses)
	reports = append(reports, wordsOut)
	charsOut := CSVChars(ctx, dir, verses)
	reports = append(reports, charsOut)
	return reports
}

func JSONTimestampHierachyRpt(ctx context.Context, dir string, verses []proofing_rpt.Verse2) db.Output {
	var output = db.Output{Component: "proofing_rpt", Report: "all_timestamps_json"}
	bytes, err := json.MarshalIndent(verses, "", "  ")
	if err != nil {
		log.Warn(ctx, err, "Could not Marshal Timestamps")
		return output
	}
	filePath := filepath.Join(dir, "all_timestamps.json")
	err = os.WriteFile(filePath, bytes, 0644)
	if err != nil {
		log.Warn(ctx, err, "Could not write file of json report")
		return output
	}
	output.FilePath = filePath
	return output
}

func CSVScripts(ctx context.Context, dir string, verses []proofing_rpt.Verse2) db.Output {
	var output = db.Output{Component: "proofing_rpt", Report: "script_timestamps_csv"}
	filePath := filepath.Join(dir, "script_timestamps.csv")
	file, err := os.Create(filePath)
	if err != nil {
		log.Warn(ctx, err, "Could not create file for script_timestamps.csv")
		return output
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush() // REQUIRED — the writer buffers
	err = writer.Write([]string{"ref", "book_id", "chapter", "verse", "begin_ts", "end_ts", "text"})
	if err != nil {
		log.Warn(ctx, err, "Could not write header to script timestamps")
		return output
	}
	for _, vs := range verses {
		var script = make([]string, 0, len(vs.Words))
		for _, wd := range vs.Words {
			script = append(script, wd.Text)
		}
		err = writer.Write([]string{
			vs.LineRef.Description(),
			vs.LineRef.BookId,
			strconv.Itoa(vs.LineRef.ChapterNum),
			vs.LineRef.VerseStr,
			strconv.FormatFloat(vs.BeginTS, 'f', 4, 64),
			strconv.FormatFloat(vs.EndTS, 'f', 4, 64),
			strings.Join(script, " "),
		})
		if err != nil {
			log.Warn(ctx, err, "Could not write a line to scripts csv")
			return output
		}
	}
	output.FilePath = filePath
	return output
}

func CSVWords(ctx context.Context, dir string, verses []proofing_rpt.Verse2) db.Output {
	var output = db.Output{Component: "proofing_rpt", Report: "word_timestamps_csv"}
	filePath := filepath.Join(dir, "word_timestamps.csv")
	file, err := os.Create(filePath)
	if err != nil {
		log.Warn(ctx, err, "Could not create file for word_timestamps.csv")
		return output
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush() // REQUIRED — the writer buffers
	err = writer.Write([]string{"ref", "book_id", "chapter", "verse", "word_seq", "begin_ts", "end_ts", "word"})
	if err != nil {
		log.Warn(ctx, err, "Could not write header to word timestamps")
		return output
	}
	for _, vs := range verses {
		for i, wd := range vs.Words {
			err = writer.Write([]string{
				vs.LineRef.Description(),
				vs.LineRef.BookId,
				strconv.Itoa(vs.LineRef.ChapterNum),
				vs.LineRef.VerseStr,
				strconv.Itoa(i + 1),
				strconv.FormatFloat(wd.BeginTS, 'f', 4, 64),
				strconv.FormatFloat(wd.EndTS, 'f', 4, 64),
				wd.Text,
			})
			if err != nil {
				log.Warn(ctx, err, "Could not write a line to words csv")
				return output
			}
		}
	}
	output.FilePath = filePath
	return output
}

func CSVChars(ctx context.Context, dir string, verses []proofing_rpt.Verse2) db.Output {
	var output = db.Output{Component: "proofing_rpt", Report: "char_timestamps_csv"}
	filePath := filepath.Join(dir, "char_timestamps.csv")
	file, err := os.Create(filePath)
	if err != nil {
		log.Warn(ctx, err, "Could not create file for char_timestamps.csv")
		return output
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush() // REQUIRED — the writer buffers
	err = writer.Write([]string{"ref", "book_id", "chapter", "verse", "word_seq", "char_seq",
		"begin_ts", "end_ts", "char"})
	if err != nil {
		log.Warn(ctx, err, "Could not write header to char timestamps")
		return output
	}
	for _, vs := range verses {
		for i, wd := range vs.Words {
			for j, ch := range wd.Chars {
				err = writer.Write([]string{
					vs.LineRef.Description(),
					vs.LineRef.BookId,
					strconv.Itoa(vs.LineRef.ChapterNum),
					vs.LineRef.VerseStr,
					strconv.Itoa(i + 1),
					strconv.Itoa(j + 1),
					strconv.FormatFloat(ch.BeginTS, 'f', 4, 64),
					strconv.FormatFloat(ch.EndTS, 'f', 4, 64),
					string(ch.Char),
				})
				if err != nil {
					log.Warn(ctx, err, "Could not write a line to words csv")
					return output
				}
			}
		}
	}
	output.FilePath = filePath
	return output
}
