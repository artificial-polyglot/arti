package script_reader

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/xuri/excelize/v2"
)

/*
This file contains methods for reading xlsx, xlsm, and xls spreadsheets.
Rather than use the extension to identify the type, it uses the first
two bytes.
*/

func ReadSheet(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 2)
	_, _ = f.Read(buf)
	_ = f.Close()

	if buf[0] == 0x50 && buf[1] == 0x4B {
		return ReadXLSXSheet(filePath)
	}
	if buf[0] == 0xD0 && buf[1] == 0xCF {
		return ReadXLSSheet(filePath)
	}
	return nil, fmt.Errorf("unrecognized spreadsheet file format: %s", filePath)

}

func ReadXLSXSheet(filePath string) ([][]string, error) {
	var rows [][]string
	file, err := excelize.OpenFile(filePath)
	if err != nil {
		return rows, err
	}
	sheets := file.GetSheetList()
	rows, err = file.GetRows(sheets[0])
	_ = file.Close()
	return rows, err
}

func ReadXLSSheet(filePath string) ([][]string, error) {
	var rows [][]string
	cmd := exec.Command("python",
		"./excel_xlrd.py",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return rows, err
	}
	err = json.Unmarshal(output, &rows)
	return rows, err
}
