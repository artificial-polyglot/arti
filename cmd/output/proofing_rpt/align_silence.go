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

// silencePositionOf classifies the gap between the char at (prevVi, prevWi)
// and the char at (vi, wi) - both are indices into verses - as within a word,
// between words, between verses, or between chapters.
func silencePositionOf(verses []Verse2, prevVi, prevWi, vi, wi int) SilencePosition {
	switch {
	case prevVi == vi && prevWi == wi:
		return betweenChars
	case prevVi == vi:
		return betweenWords
	case verses[prevVi].LineRef.BookId == verses[vi].LineRef.BookId && verses[prevVi].LineRef.ChapterNum == verses[vi].LineRef.ChapterNum:
		return betweenVerses
	default:
		return betweenChapters
	}
}

func (a *AlignSilence) Process() ([]Verse2, map[string]generic.AudioFile, *log.Status) {
	var audioURLs map[string]generic.AudioFile
	verses, status := SelectCharData(a.conn, FA_SCORE_CUTOFF)
	if status != nil {
		return verses, audioURLs, status
	}
	var charSilence, wordSilence, verseSilence, chapterSilence []float64
	var prev *Char2
	var prevVi, prevWi int
	for vi := range verses {
		for wi := range verses[vi].Words {
			chars := verses[vi].Words[wi].Chars
			for ci := range chars {
				curr := &chars[ci]
				if prev != nil {
					switch silencePositionOf(verses, prevVi, prevWi, vi, wi) {
					case betweenChars:
						prev.Silence = curr.BeginTS - prev.EndTS
						charSilence = append(charSilence, prev.Silence)
					case betweenWords:
						prev.Silence = curr.BeginTS - prev.EndTS
						wordSilence = append(wordSilence, prev.Silence)
					case betweenVerses:
						// Char timestamps are per-verse (0-based within that verse's
						// chopped audio clip), so prev and curr live in different
						// clips. Verse.BeginTS is the chapter-absolute offset used to
						// chop each clip, so add it back in to compare in one frame.
						prev.Silence = (verses[vi].BeginTS + curr.BeginTS) - (verses[prevVi].BeginTS + prev.EndTS)
						verseSilence = append(verseSilence, prev.Silence)
					case betweenChapters:
						var duration float64
						duration, status = a.SelectDuration(verses[prevVi].ScriptId)
						if status != nil {
							return verses, audioURLs, status
						}
						prev.Silence = duration - prev.EndTS
						if prev.Silence < 0 {
							prev.Silence = 0
						}
						chapterSilence = append(chapterSilence, prev.Silence)
					}
				}
				prev = curr
				prevVi, prevWi = vi, wi
			}
		}
	}
	mean, stddev := a.analyzeData(charSilence)
	var charLimit = mean + (0.0 * stddev)
	mean, stddev = a.analyzeData(wordSilence)
	var wordLimit = mean + (0.0 * stddev)
	mean, stddev = a.analyzeData(verseSilence)
	var verseLimit = mean + (0.0 * stddev)
	mean, stddev = a.analyzeData(chapterSilence)
	var chapLimit = mean + (0.0 * stddev)
	a.markSilenceOutliers(verses, charLimit, wordLimit, verseLimit, chapLimit)
	verses, status = a.CompareLines2ASR(verses)
	if status != nil {
		return verses, audioURLs, status
	}
	audioURLs, status = db.CreateAudioFileMap(a.conn)
	a.ComputeOpacity(verses, OPACITY_CUTOFF)
	return verses, audioURLs, status
}

func (a *AlignSilence) analyzeData(data []float64) (mean, stddev float64) {
	if len(data) == 0 {
		return 0.0, 0.0
	}
	mean = stat.Mean(data, nil)
	stddev = stat.StdDev(data, nil)
	return mean, stddev
}

func (a *AlignSilence) markSilenceOutliers(verses []Verse2, charLimit, wordLimit, verseLimit, chapLimit float64) {
	var prev *Char2
	var prevVi, prevWi int
	for vi := range verses {
		for wi := range verses[vi].Words {
			chars := verses[vi].Words[wi].Chars
			for ci := range chars {
				curr := &chars[ci]
				if prev != nil {
					switch silencePositionOf(verses, prevVi, prevWi, vi, wi) {
					case betweenChars:
						if prev.Silence >= charLimit {
							prev.SilenceLong = int(betweenCharsLong)
						}
					case betweenWords:
						if prev.Silence >= wordLimit {
							prev.SilenceLong = int(betweenWordsLong)
						}
					case betweenVerses:
						if prev.Silence >= verseLimit {
							prev.SilenceLong = int(betweenVersesLong)
						}
					case betweenChapters:
						if prev.Silence >= chapLimit {
							prev.SilenceLong = int(betweenChaptersLong)
						}
					}
				}
				prev = curr
				prevVi, prevWi = vi, wi
			}
		}
	}
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
