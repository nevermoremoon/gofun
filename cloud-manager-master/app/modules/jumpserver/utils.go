package jumpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tealeg/xlsx"
	"net/url"
	"strconv"
	"strings"
)

func buildHttpQuery(paramMap map[string]interface{}) string {
	excludeMap := map[string]bool{"Action": true}
	var uri url.URL
	query := uri.Query()
	for k, v := range paramMap {
		vString := fmt.Sprint(v)
		if excludeMap[k] || vString == "" {
			continue
		}
		query.Add(strings.ToLower(k), vString)
	}
	return query.Encode()
}

func convertToStruct(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err == nil {
		err = json.Unmarshal(b, out)
	}
	return err
}

type HeaderColumn struct {
	Field string
	Title string
}
func Export(filePath string, sheetName string, hosts []*AssetsHost) (err error){
	file := xlsx.NewFile()
	sheet, err := file.AddSheet(sheetName)
	if err != nil {
		return err
	}
	headers := []*HeaderColumn {
		{
			Field: "ID",
			Title: "编号",
		},
		{
			Field: "Host",
			Title: "主机名",
		},
		{
			Field: "IP",
			Title: "IP",
		},
		{
			Field: "Comment",
			Title: "备注",
		},
	}
	style := map[string]float64 {
		"ID":      0.8,
		"Host":    3.0,
		"IP":      2.0,
		"Comment": 7.0,
	}
	sheet, _ = setHeader(sheet, headers, style)
	for i, host := range hosts {
		data := make(map[string]string)
		data["ID"] = strconv.Itoa(i+1)
		data["Host"] = host.Hostname
		data["IP"] = host.Ip
		data["Comment"] = host.Comment
		row := sheet.AddRow()
		row.SetHeightCM(0.8)
		for _, field := range headers {
			row.AddCell().Value= data[field.Field]
		}
	}
	err = file.Save(filePath)
	return err
}

func setHeader(sheet *xlsx.Sheet, header []*HeaderColumn, width map[string]float64) (*xlsx.Sheet, error) {
	if len(header) <= 0 {
		return nil, errors.New("Excel.SetHeader head can't empty")
	}
	style := xlsx.NewStyle()
	font := xlsx.DefaultFont()
	font.Bold = true
	alignment := xlsx.DefaultAlignment()
	alignment.Vertical = "center"
	style.Font = *font
	style.Alignment = *alignment

	row := sheet.AddRow()
	row.SetHeightCM(1.0)
	row_w := make([]string, 0)
	for _, column := range header {
		row_w = append(row_w, column.Field)
		cell := row.AddCell()
		cell.Value = column.Title
		cell.SetStyle(style)
	}
	if len(row_w) > 0 {
		for k, v := range row_w {
			if width[v] > 0.0 {
				_ = sheet.SetColWidth(k, k, width[v]*10)
			}
		}
	}
	return sheet, nil
}