package proofing_rpt

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"gonum.org/v1/gonum/stat"
)

type ErrorType int

const (
	noError ErrorType = iota
	scoreCritical
	scoreQuestion
	betweenCharsLong
	betweenWordsLong
	betweenVersesLong
	betweenChaptersLong
)

type SilencePosition int

const (
	betweenChars SilencePosition = iota + 1
	betweenWords
	betweenVerses
	betweenChapters
)

const FA_SCORE_CUTOFF = 0.5
const OPACITY_CUTOFF = 0.5

type AlignSilence struct {
	ctx  context.Context
	conn db.DBAdapter
}

func NewAlignSilence(conn db.DBAdapter) AlignSilence {
	var a AlignSilence
	a.ctx = conn.Ctx
	a.conn = conn
	return a
}

func (a *AlignSilence) Process() ([]Verse2, map[string]generic.AudioFile, *log.Status) {
	var verses []Verse2
	var audioURLs map[string]generic.AudioFile
	faChars, status := a.conn.SelectFACharTimestamps(FA_SCORE_CUTOFF)
	if status != nil {
		return verses, audioURLs, status
	}
	for i := 0; i < len(faChars)-1; i++ {
		var curr = faChars[i]
		var next = faChars[i+1]
		faChars[i].Duration = curr.EndTS - curr.BeginTS
		faChars[i].Silence = next.BeginTS - curr.EndTS
		if curr.WordId == next.WordId {
			faChars[i].SilencePos = int(betweenChars)
		} else if curr.LineId == next.LineId {
			faChars[i].SilencePos = int(betweenWords)
		} else if curr.LineRef.BookId == next.LineRef.BookId && curr.LineRef.ChapterNum == next.LineRef.ChapterNum {
			faChars[i].SilencePos = int(betweenVerses)
		} else {
			faChars[i].SilencePos = int(betweenChapters)
			var duration float64
			duration, status = a.SelectDuration(faChars[i].LineId)
			if status != nil {
				return verses, audioURLs, status
			}
			if duration > curr.EndTS {
				faChars[i].Silence = duration - curr.EndTS
			} else {
				faChars[i].Silence = 0.0
			}
		}
	}
	mean, stddev := a.analyzeData(a.getDurations(faChars))
	//fmt.Println("Char Widths:", mean, stddev, mini, maxi)
	mean, stddev = a.analyzeData(a.getSilence(faChars, betweenChars))
	//fmt.Println("Between Chars:", mean, stddev, mini, maxi)
	var charLimit = mean + (4.0 * stddev)
	mean, stddev = a.analyzeData(a.getSilence(faChars, betweenWords))
	//fmt.Println("Between Words:", mean, stddev, mini, maxi)
	var wordLimit = mean + (4.0 * stddev)
	mean, stddev = a.analyzeData(a.getSilence(faChars, betweenVerses))
	//fmt.Println("Between Verses:", mean, stddev, mini, maxi)
	var verseLimit = mean + (4.0 * stddev)
	mean, stddev = a.analyzeData(a.getSilence(faChars, betweenChapters))
	//fmt.Println("Between Chapters:", mean, stddev, mini, maxi)
	var chapLimit = mean + (3.0 * stddev)
	a.markSilenceOutliers(faChars, charLimit, wordLimit, verseLimit, chapLimit)
	verses = a.PrepareDataForWriter(faChars)
	verses, status = a.CompareLines2ASR(verses)
	if status != nil {
		return verses, audioURLs, status
	}
	audioURLs, status = db.CreateAudioFileMap(a.conn)
	a.ComputeOpacity(verses, OPACITY_CUTOFF)
	return verses, audioURLs, status
}

func (a *AlignSilence) getDurations(chars []generic.AlignChar) []float64 {
	var data []float64
	for _, ch := range chars {
		data = append(data, ch.Duration)
	}
	return data
}

func (a *AlignSilence) getSilence(chars []generic.AlignChar, pos SilencePosition) []float64 {
	var data []float64
	posInt := int(pos)
	for _, ch := range chars {
		if ch.SilencePos == posInt {
			data = append(data, ch.Silence)
		}
	}
	return data
}

func (a *AlignSilence) analyzeData(data []float64) (mean, stddev float64) {
	if len(data) == 0 {
		return 0.0, 0.0
	}
	mean = stat.Mean(data, nil)
	stddev = stat.StdDev(data, nil)
	return mean, stddev
}

func (a *AlignSilence) markSilenceOutliers(chars []generic.AlignChar, charLimit, wordLimit, verseLimit, chapLimit float64) { //, mean float64, stddev float64,
	for i, ch := range chars {
		switch SilencePosition(ch.SilencePos) {
		case betweenChars:
			if ch.Silence >= charLimit {
				chars[i].SilenceLong = int(betweenCharsLong)
			}
		case betweenWords:
			if ch.Silence >= wordLimit {
				chars[i].SilenceLong = int(betweenWordsLong)
			}
		case betweenVerses:
			if ch.Silence >= verseLimit {
				chars[i].SilenceLong = int(betweenVersesLong)
			}
		case betweenChapters:
			if ch.Silence >= chapLimit {
				chars[i].SilenceLong = int(betweenChaptersLong)
			}
		}
	}
}

func (a *AlignSilence) groupByLine(chars []generic.AlignChar) []generic.AlignLine {
	var result []generic.AlignLine
	if len(chars) == 0 {
		return result
	}
	currRef := chars[0].LineRef
	start := 0
	for i, ch := range chars {
		if ch.LineRef != currRef { // compare on lineRef makes verse a unique key
			currRef = ch.LineRef
			oneLine := make([]generic.AlignChar, i-start)
			copy(oneLine, chars[start:i])
			start = i
			var line generic.AlignLine
			line.Chars = oneLine
			result = append(result, line)
		}
	}
	if start < len(chars) {
		lastLine := make([]generic.AlignChar, len(chars)-start)
		copy(lastLine, chars[start:])
		var line generic.AlignLine
		line.Chars = lastLine
		result = append(result, line)
	}
	count := 0
	for _, l := range result {
		count += len(l.Chars)
	}
	if count != len(chars) {
		log.Warn(a.ctx, "Original chars", len(chars), "final chars", count)
	}
	return result
}

func (a *AlignSilence) countErrors(lines []generic.AlignLine) {
	var total int
	var critScoreError int
	var questScoreError int
	var count = make([]int, 8)
	for _, line := range lines {
		for _, ch := range line.Chars {
			total++
			if ch.ScoreError == int(scoreCritical) {
				critScoreError++
			} else if ch.ScoreError == int(scoreQuestion) {
				questScoreError++
			}
			count[ch.SilenceLong]++
		}
	}
	fmt.Println("NO Error\t", count[noError]-critScoreError-questScoreError)
	fmt.Println("ScoreCritical", critScoreError)
	fmt.Println("ScoreQuestion", questScoreError)
	fmt.Println("BetweenCharsLong", count[betweenCharsLong])
	fmt.Println("BetweenWordsLong", count[betweenWordsLong])
	fmt.Println("BetweenVersesLong", count[betweenVersesLong])
	fmt.Println("BetweenChaptersLong", count[betweenChaptersLong])
	fmt.Println("Total\t", total)
}

func (a *AlignSilence) SelectDuration(scriptId int64) (float64, *log.Status) {
	query := `SELECT script_end_ts FROM scripts WHERE script_id = ?`
	row := a.conn.DB.QueryRow(query, scriptId)
	var timestamp float64
	err := row.Scan(&timestamp)
	if err == sql.ErrNoRows {
		return 0.0, nil
	} else if err != nil {
		return 0.0, log.Error(a.ctx, 500, err, "Failed to select from qa_align_scripts")
	}
	return timestamp, nil
}

func (p *AlignSilence) ComputeOpacity(verses []Verse2, opacityCutoff float64) {
	for i := range verses {
		for j := range verses[i].Words {
			if verses[i].Words[j].FAScore < opacityCutoff {
				verses[i].Words[j].Opacity = 1.0 - verses[i].Words[j].FAScore/opacityCutoff
			}
		}
	}
}
