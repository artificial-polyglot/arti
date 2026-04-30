package read

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
	log "github.com/faithcomesbyhearing/fcbh-dataset-io/logger"
)

func TestUSFMParser(t *testing.T) {
	ctx := context.Background()
	var directory = "test_data"
	var files []input.InputFile
	file1 := input.InputFile{BookId: "LUK", Directory: directory, Filename: "43LUKCFM.SFM", MediaType: "text/plain"}
	//file2 := input.InputFile{BookId: "LUK", Directory: directory, Filename: "43LUKDWK.SFM", MediaType: "text/plain"}
	files = append(files, file1)
	var database = directory + "/usfm_test.db"
	db.DestroyDatabase(database)
	var conn = db.NewDBAdapter(ctx, database)
	parser := NewUSFMParser(conn)
	status := parser.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
	selectScripts(conn, t)
	conn.Close()
}

func selectScripts(conn db.DBAdapter, t *testing.T) {
	query := `SELECT script_id, book_id, chapter_num, verse_num, verse_str, usfm_style, script_text
		FROM scripts ORDER BY script_id`
	rows, err := conn.DB.Query(query)
	if err != nil {
		t.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		var rec db.Script
		err = rows.Scan(&rec.ScriptId, &rec.BookId, &rec.ChapterNum,
			&rec.VerseNum, &rec.VerseStr, &rec.UsfmStyle, &rec.ScriptText)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(rec.ChapterNum, rec.VerseStr, "usfm:", rec.UsfmStyle, "[", rec.ScriptText, "]")
	}
	err = rows.Err()
	if err != nil {
		t.Error(err)
	}
}

func TestUSXUSFMCompare(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	const compareBucket = "pretest-audio"

	usxPrefixes := []string{
		"Uploaded/N2HOYWFW Holiya [T] (HOY)/N2HOYWFW Text/USX/",
		//"N2XISWIN Kisan (XIS)/N2XISWIN Text/USX/",
		//"2025-09-08/O2NHEWYI O2_Nahuatl Huasteca, Eastern (NHE)/O2NHEWYI Text/O2NHEWYI USX/",
		//"O2NHEWYI Nahuatl Huasteca, Eastern (NHE)/O2NHEWYI Text/O2NHEWYI USX/",
		//"2025-09-05/N2BHIWFW Bhilali (BHI)/N2BHIWFW Text/N2BHIWFW USX/",
	}

	client, err := newCompareClient(ctx)
	if err != nil {
		t.Fatal("Failed to create S3 client:", err)
	}

	tmpDir := filepath.Join(os.Getenv("HOME"), "FCBH2024", "tmp")
	usxSubDir := filepath.Join(tmpDir, "usx")
	sfmSubDir := filepath.Join(tmpDir, "sfm")
	usxDBPath := filepath.Join(tmpDir, "usx.db")
	sfmDBPath := filepath.Join(tmpDir, "sfm.db")

	for _, usxPrefix := range usxPrefixes {
		parentPrefix := path.Dir(strings.TrimSuffix(usxPrefix, "/")) + "/"

		sfmPrefix, status := findSFMDir(ctx, client, compareBucket, parentPrefix)
		if status != nil {
			t.Fatalf("Failed to find SFM dir for %s: %v", usxPrefix, status)
		}

		usxFiles, status := downloadToSubDir(ctx, client, compareBucket, usxPrefix, usxSubDir, "")
		if status != nil {
			t.Fatalf("Failed to download USX files from %s: %v", usxPrefix, status)
		}

		sfmFiles, status := downloadToSubDir(ctx, client, compareBucket, sfmPrefix, sfmSubDir, ".sfm")
		if status != nil {
			t.Fatalf("Failed to download SFM files from %s: %v", sfmPrefix, status)
		}

		os.Remove(sfmDBPath)
		sfmConn := db.NewDBAdapter(ctx, sfmDBPath)
		sfmParser := NewUSFMParser(sfmConn)
		status = sfmParser.ProcessFiles(sfmFiles)
		sfmConn.Close()
		if status != nil {
			t.Fatalf("SFM parse failed for %s: %v", sfmPrefix, status)
		}

		os.Remove(usxDBPath)
		usxConn := db.NewDBAdapter(ctx, usxDBPath)
		usxParser := NewUSXParser(usxConn)
		status = usxParser.ProcessFiles(usxFiles)
		usxConn.Close()
		if status != nil {
			t.Fatalf("USX parse failed for %s: %v", usxPrefix, status)
		}

		diffs, status := compareScriptDatabases(ctx, usxDBPath, sfmDBPath)
		if status != nil {
			t.Fatalf("Comparison failed for %s: %v", usxPrefix, status)
		}
		if len(diffs) > 0 {
			t.Logf("DIFFERENCES in %s <-> %s:", usxPrefix, sfmPrefix)
			for _, d := range diffs {
				t.Logf("  [%s] ch=%d v=%d style=%q text=%q", d.source, d.chapterNum, d.verseNum, d.usfmStyle, d.scriptText)
			}
			t.FailNow()
		}
	}
}

func newCompareClient(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "us-west-2"
	}), nil
}

// findSFMDir lists objects under parentPrefix and returns the S3 prefix of the
// first directory containing .sfm files.
func findSFMDir(ctx context.Context, client *s3.Client, bucket, parentPrefix string) (string, *log.Status) {
	sfmDirs := make(map[string]bool)
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(parentPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", log.Error(ctx, 500, err, "Failed to list objects under", parentPrefix)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if strings.HasSuffix(strings.ToLower(key), ".sfm") {
				sfmDirs[path.Dir(key)+"/"] = true
			}
		}
	}
	if len(sfmDirs) == 0 {
		return "", log.ErrorNoErr(ctx, 404, "No SFM directory found under", parentPrefix)
	}
	for dir := range sfmDirs {
		return dir, nil
	}
	return "", nil
}

// downloadToSubDir clears localDir, downloads all objects under s3Prefix into it,
// and returns an InputFile slice with Directory, Filename, and BookId populated.
func downloadToSubDir(ctx context.Context, client *s3.Client, bucket, s3Prefix, localDir, extFilter string) ([]input.InputFile, *log.Status) {
	if err := os.RemoveAll(localDir); err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to remove directory", localDir)
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to create directory", localDir)
	}
	var files []input.InputFile
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(s3Prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, log.Error(ctx, 500, err, "Failed to list objects under", s3Prefix)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			filename := path.Base(key)
			if extFilter != "" && strings.ToLower(path.Ext(filename)) != extFilter {
				continue
			}
			localPath := filepath.Join(localDir, filename)
			resp, err := client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return nil, log.Error(ctx, 500, err, "Failed to get object", key)
			}
			f, ferr := os.Create(localPath)
			if ferr != nil {
				resp.Body.Close()
				return nil, log.Error(ctx, 500, ferr, "Failed to create file", localPath)
			}
			_, copyErr := io.Copy(f, resp.Body)
			resp.Body.Close()
			f.Close()
			if copyErr != nil {
				return nil, log.Error(ctx, 500, copyErr, "Failed to write file", localPath)
			}
			files = append(files, input.InputFile{
				Directory: localDir,
				Filename:  filename,
				BookId:    bookIdForCompare(filename),
			})
		}
	}
	return files, nil
}

// bookIdForCompare extracts the 3-character USFM book ID from a USX or SFM filename.
// Handles "001GEN.usx" / "001GEN.sfm" (length 10) and "GEN.usx" / "GEN.sfm" (length 7).
func bookIdForCompare(filename string) string {
	switch len(filename) {
	case 7:
		return filename[0:3]
	default:
		return filename[2:5]
	}
}

type scriptDiff struct {
	source     string
	chapterNum int
	verseNum   int
	usfmStyle  string
	scriptText string
}

// compareScriptDatabases ATTACHes both db files to an in-memory connection and runs
// a symmetric difference query, returning rows present in one but not the other.
func compareScriptDatabases(ctx context.Context, usxDBPath, sfmDBPath string) ([]scriptDiff, *log.Status) {
	cmp, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to open comparison database")
	}
	defer cmp.Close()

	if _, err = cmp.Exec(fmt.Sprintf("ATTACH '%s' AS usx", usxDBPath)); err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to attach USX database", usxDBPath)
	}
	if _, err = cmp.Exec(fmt.Sprintf("ATTACH '%s' AS sfm", sfmDBPath)); err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to attach SFM database", sfmDBPath)
	}

	query := `
		SELECT 'only_in_usx' AS source, chapter_num, verse_num, usfm_style, script_text
		 FROM usx.scripts
		 EXCEPT
		 SELECT 'only_in_usx', chapter_num, verse_num, usfm_style, script_text
		 FROM sfm.scripts
		UNION ALL
		SELECT 'only_in_sfm' AS source, chapter_num, verse_num, usfm_style, script_text
		 FROM sfm.scripts
		 EXCEPT
		 SELECT 'only_in_sfm', chapter_num, verse_num, usfm_style, script_text
		 FROM usx.scripts`

	rows, err := cmp.Query(query)
	if err != nil {
		return nil, log.Error(ctx, 500, err, "Failed to run comparison query")
	}
	defer rows.Close()

	var diffs []scriptDiff
	for rows.Next() {
		var d scriptDiff
		if err = rows.Scan(&d.source, &d.chapterNum, &d.verseNum, &d.usfmStyle, &d.scriptText); err != nil {
			return nil, log.Error(ctx, 500, err, "Failed to scan diff row")
		}
		diffs = append(diffs, d)
	}
	return diffs, nil
}
