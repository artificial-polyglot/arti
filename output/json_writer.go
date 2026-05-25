package output

import (
	"bufio"
	"encoding/json"
	log "github.com/artificial-polyglot/arti/logger"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

func (o *Output) WriteJSON(structs []any, meta []Meta) (string, *log.Status) {
	var filename string
	var status *log.Status
	file, err := os.Create(filepath.Join(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+".json"))
	//file, err := os.CreateTemp(os.Getenv(`FCBH_DATASET_TMP`), o.requestName+"_*.json")
	if err != nil {
		status = log.Error(o.ctx, 500, err, `failed to create temp file`)
		return filename, status
	}
	filename = file.Name()
	writer := bufio.NewWriter(file)
	var names []string
	for _, mt := range meta {
		if mt.Cols == 1 {
			names = append(names, mt.Tag)
		} else {
			for i := 0; i < mt.Cols; i++ {
				names = append(names, mt.Tag+strconv.Itoa(i))
			}
		}
	}
	var resp = make([]map[string]any, 0, len(structs))
	for _, scr := range structs {
		str := reflect.ValueOf(scr)
		var rec = make(map[string]any)
		for _, mt := range meta {
			data := str.Field(mt.Index)
			if data.Kind() == reflect.Slice {
				for i := 0; i < data.Len(); i++ {
					item := data.Index(i)
					if item.Kind() == reflect.Slice {
						for j := 0; j < item.Len(); j++ {
							rec[names[mt.CSVPos+j]] = o.ToValue(item.Index(j))
						}
						if i < data.Len()-1 {
							resp = append(resp, rec)
							rec = make(map[string]any)
						}
					} else {
						rec[names[mt.CSVPos+i]] = o.ToValue(item)
					}
				}
			} else {
				rec[names[mt.CSVPos]] = o.ToValue(data)
			}
		}
		resp = append(resp, rec)
		rec = make(map[string]any)
	}
	var encoder = json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(resp)
	if err != nil {
		status = log.Error(o.ctx, 500, err, `failed to write json response`)
		return filename, status
	}
	err = writer.Flush()
	if err != nil {
		status = log.Error(o.ctx, 500, err, `failed to flush json response`)
	}
	err = file.Close()
	if err != nil {
		status = log.Error(o.ctx, 500, err, `error closing json file`)
	}
	return filename, status
}

func (o *Output) ToValue(value reflect.Value) any {
	var result any
	switch value.Kind() {
	case reflect.Bool:
		result = value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		result = value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result = value.Uint()
	case reflect.Float32:
		result = value.Float()
	case reflect.Float64:
		result = value.Float()
	case reflect.String:
		result = value.String()
	default:
		panic(`output.ToValue() cannot convert value of type` + value.Type().String())
	}
	return result
}
