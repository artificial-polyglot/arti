package output

import (
	"encoding/csv"
	log "github.com/artificial-polyglot/arti/logger"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

func (o *Output) WriteCSV(structs []any, meta []Meta) (string, *log.Status) {
	var filename string
	file, err := os.Create(filepath.Join(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+".csv"))
	//file, err := os.CreateTemp(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+"_*.csv")
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, `failed to create temp file`)
	}
	filename = file.Name()
	writer := csv.NewWriter(file)
	var header []string
	for _, mt := range meta {
		if mt.Cols == 1 {
			header = append(header, mt.Tag)
		} else {
			for i := 0; i < mt.Cols; i++ {
				header = append(header, mt.Tag+strconv.Itoa(i))
			}
		}
	}
	_ = writer.Write(header)
	for _, scr := range structs {
		str := reflect.ValueOf(scr)
		var line = make([]string, len(header))
		for _, mt := range meta {
			data := str.Field(mt.Index)
			if data.Kind() == reflect.Slice {
				for i := 0; i < data.Len(); i++ {
					if data.Index(i).Kind() == reflect.Slice {
						for j := 0; j < data.Index(i).Len(); j++ {
							line[mt.CSVPos+j] = o.ToString(data.Index(i).Index(j))
						}
						if i < data.Len()-1 {
							_ = writer.Write(line)
							line = make([]string, len(header))
						}
					} else {
						line[mt.CSVPos+i] = o.ToString(data.Index(i))
					}
				}
			} else {
				line[mt.CSVPos] = o.ToString(data)
			}
		}
		_ = writer.Write(line)
		line = make([]string, len(header))
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, `failed to flush csv file`)
	}
	err = file.Close()
	if err != nil {
		return filename, log.Error(o.ctx, 500, err, `error closing json file`)
	}
	return filename, nil
}

// ToString converts scalar values to string.  It does not convert the following kind.
// reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Struct, reflect.Array,
// reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid
func (o *Output) ToString(value reflect.Value) string {
	var result string
	switch value.Kind() {
	case reflect.Bool:
		result = strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		result = strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result = strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32:
		result = strconv.FormatFloat(value.Float(), 'f', -1, 32)
	case reflect.Float64:
		result = strconv.FormatFloat(value.Float(), 'f', -1, 32)
	case reflect.String:
		result = value.String()
	default:
		panic(`output.ToString() cannot convert value of type` + value.Type().String())
	}
	return result
}
