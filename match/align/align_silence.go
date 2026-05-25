package align

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/utility/ffmpeg"
	"gonum.org/v1/gonum/stat"
	"strconv"
	"strings"
)

const (
	criticalThreshold = 0.0001 // 0.001
	questionThreshold = 0.001  // 0.001
	//silenceStdevs     = 4.0    // intended to make it rare
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

type AlignSilence struct {
	ctx     context.Context
	conn    db.DBAdapter
	asrConn db.DBAdapter
}

func NewAlignSilence(ctx context.Context, conn db.DBAdapter, asrConn db.DBAdapter) AlignSilence {
	var a AlignSilence
	a.ctx = ctx
	a.conn = conn
	a.asrConn = asrConn
	return a
}

func (a *AlignSilence) Process(audioDirectory string) ([]generic.AlignLine, string, *log.Status) {
	var faLines []generic.AlignLine
	var status *log.Status
	faChars, status := a.conn.SelectFACharTimestamps()
	if status != nil {
		return faLines, "", status
	}
	for i := 0; i < len(faChars)-1; i++ {
		faChars[i].Duration = faChars[i].EndTS - faChars[i].BeginTS
		var curr = faChars[i]
		var next = faChars[i+1]
		faChars[i].Silence = faChars[i+1].BeginTS - faChars[i].EndTS
		if curr.WordId == next.WordId {
			faChars[i].SilencePos = int(betweenChars)
		} else if curr.LineId == next.LineId {
			faChars[i].SilencePos = int(betweenWords)
		} else if curr.AudioFile == next.AudioFile {
			faChars[i].SilencePos = int(betweenVerses)
		} else {
			faChars[i].SilencePos = int(betweenChapters)
			var duration float64
			duration, status = ffmpeg.GetAudioDuration(a.ctx, audioDirectory, faChars[i].AudioFile)
			faChars[i].Silence = duration - curr.EndTS
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
	faLines = a.groupByLine(faChars)
	faLines, status = a.compareLines2ASR(faLines, a.asrConn)
	if status != nil {
		return faLines, "", status
	}
	filenameMap, status := a.generateBookChapterFilenameMap()
	//a.countErrors(faLines)
	return faLines, filenameMap, status
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
	return result
}

func (a *AlignSilence) generateBookChapterFilenameMap() (string, *log.Status) {
	chapters, status := a.conn.SelectBookChapterFilename()
	if status != nil {
		return "", status
	}
	var result []string
	result = append(result, "let fileMap = {\n")
	for i, ch := range chapters {
		key := ch.BookId + strconv.Itoa(ch.ChapterNum)
		result = append(result, "\t'"+key+"': '"+ch.AudioFile+"'")
		if i < len(chapters)-1 {
			result = append(result, ",\n")
		} else {
			result = append(result, "};\n")
		}
	}
	return strings.Join(result, ""), status
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
