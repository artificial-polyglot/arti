package fa_score_analysis

import (
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"gonum.org/v1/gonum/stat"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

func FAScoreAnalysis(conn db.DBAdapter) (string, *log.Status) {
	var status *log.Status
	var project = filepath.Base(conn.Database)
	var ext = filepath.Ext(project)
	project = project[:len(project)-len(ext)]
	var filename = project + "_fa_error.txt"
	file, err := os.Create(filename)
	if err != nil {
		return filename, log.Error(conn.Ctx, 500, err, "unable to create fa_error.txt")
	}
	defer file.Close()
	write(file, project, " FA Error Analysis\n")

	faErrors, status := getFAErrors(conn)
	if status != nil {
		return filename, status
	}
	mean := stat.Mean(faErrors, nil)
	write(file, "Mean: ", strconv.FormatFloat(mean, 'f', 2, 64))
	stdDev := stat.StdDev(faErrors, nil)
	write(file, "StdDev: ", strconv.FormatFloat(stdDev, 'f', 2, 64))

	var mini = math.Inf(1)
	var maxi = 0.0
	for _, er := range faErrors {
		if er < mini {
			mini = er
		}
		if er > maxi {
			maxi = er
		}
	}
	write(file, "Minimum: ", strconv.FormatFloat(mini, 'f', 2, 64))
	write(file, "Maximum: ", strconv.FormatFloat(maxi, 'f', 2, 64))
	// Skewness (asymmetry of distribution)
	skewness := stat.Skew(faErrors, nil)
	write(file, "Skewness: ", strconv.FormatFloat(skewness, 'f', 2, 64))
	// Kurtosis (shape of distribution)
	kurtosis := stat.ExKurtosis(faErrors, nil)
	write(file, "Kurtosis: ", strconv.FormatFloat(kurtosis, 'f', 2, 64))
	// Percentile
	write(file, "\nPercentiles")
	sort.Float64s(faErrors)
	for _, percent := range []float64{0.990, 0.991, 0.992, 0.993, 0.994, 0.995, 0.996, 0.997, 0.998,
		0.999, 0.9991, 0.9992, 0.9993, 0.9994, 0.9995, 0.9996, 0.9997, 0.9998, 0.9999} {
		percentile := stat.Quantile(percent, stat.Empirical, faErrors, nil)
		percentStr := strconv.FormatFloat((percent * 100.0), 'f', 2, 64)
		write(file, "Percentile ", percentStr, ": ", strconv.FormatFloat(percentile, 'f', 2, 64))
	}
	// Histogram
	write(file, "\nHISTOGRAM")
	var histogram = make(map[int]int)
	for _, er := range faErrors {
		histogram[int(er)]++
	}
	var keys []int
	for k := range histogram {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	numFAError := len(faErrors)
	for _, cat := range keys {
		pct := float64(histogram[cat]) / float64(numFAError) * 100.0
		write(file, "Cat: ", strconv.Itoa(cat), "-", strconv.Itoa(cat+1), " = ", strconv.FormatFloat(pct, 'f', 4, 64), "%")
	}
	return filename, status
}

func getFAErrors(conn db.DBAdapter) ([]float64, *log.Status) {
	var faErrors []float64
	var status *log.Status
	chars, status := conn.SelectFACharTimestamps()
	if status != nil {
		return faErrors, status
	}
	for _, char := range chars {
		faErrors = append(faErrors, -math.Log10(char.FAScore))
	}
	return faErrors, status
}

func write(file *os.File, args ...string) {
	for _, arg := range args {
		_, _ = file.WriteString(arg)
	}
	_, _ = file.WriteString("\n")
}
