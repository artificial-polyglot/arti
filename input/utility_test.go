package input

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/request"
)

func TestUtility_validateBookId(t *testing.T) {
	ctx := context.Background()
	bookId, status := validateBookId(ctx, "TTL")
	if status != nil {
		t.Error(status)
	}
	if bookId != "TIT" {
		t.Error(bookId, "should have been revised to TIT")
	}
}

func TestUtility_parseFilenames(t *testing.T) {
	ctx := context.Background()
	test1 := InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "001GEN.usx"}
	status := parseFilenames(ctx, &test1)
	if status != nil {
		t.Error(status)
	}
	if test1.MediaId != "DEF" {
		t.Error("Media ID should be DEF")
	}
	if test1.BookId != "GEN" {
		t.Error("Book ID should be GEN")
	}
	if test1.BookSeq != "001" && test1.BookSeq != "1" {
		t.Error("Book Seq should be 001")
	}
	test2 := InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "GEN.usx"}
	status = parseFilenames(ctx, &test2)
	if status != nil {
		t.Error(status)
	}
	if test2.BookId != "GEN" {
		t.Error("Book ID should be GEN")
	}
	if test2.BookSeq != "1" {
		t.Error("Book Seq should be 1")
	}
	test3 := InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "1GEN.usx"}
	status = parseFilenames(ctx, &test3)
	if test3.BookId != "GEN" {
		t.Error("For 1GEN.usx  book_id=GEN")
	}
	if test3.BookSeq != "1" {
		t.Error("For 1GEN.usx  bookSeq should be 1")
	}
}

func TestUtility_FindTextBookId(t *testing.T) {
	bookId := findTextBookId("0231TITXX.sfm")
	if bookId != "TIT" {
		t.Error("For 0231TITXX.sfm bookId = TIT")
	}
	bookId = findTextBookId("0231TIXX.sfm")
	if bookId != "1TI" {
		t.Error("For 0231TIXX.sfm bookId = 1TI")
	}
	bookId = findTextBookId("01231TIT.sfm")
	if bookId != "TIT" {
		t.Error("For 0231TIT.sfm bookId = TIT")
	}
	bookId = findTextBookId("01231TI.sfm")
	if bookId != "1TI" {
		t.Error("For 01231TI.sfm bookId = 1TI")
	}
	fmt.Println(bookId)
}

func TestUtility_FindAudioBookId(t *testing.T) {
	bookId, chapter, err := findAudioBookId(strings.Split("N2_QAE_BSP_006_MAT_006_VOX.mp3", "_"))
	if bookId != "MAT" {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 bookId = MAT")
	}
	if chapter != 6 {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 chapter = 6")
	}
	if err != nil {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 err = nil")
	}
}
