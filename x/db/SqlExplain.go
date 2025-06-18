package db

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type SqlExplain struct {
	dbType string
	result sqlResult
}

type sqlResult = []map[string]any

var def_fields = []string{"id", "select_type", "table", "partitions", "type", "possible_keys", "key", "key_len", "ref", "rows", "filtered", "Extra"}

func (this *SqlExplain) DrawConsole() { /*{{{*/
	arr_max_length := []int{}
	records := map[int][]string{}
	fields := def_fields

	if this.dbType != "mysql" {
		for _, record := range this.result {
			fields = []string{}
			for field, _ := range record {
				fields = append(fields, field)
			}

			break
		}
	}

	for _, v := range fields {
		arr_max_length = append(arr_max_length, len(v)+2)
	}

	for i, record := range this.result {
		records[i] = []string{}

		for j, v := range fields {
			if record[v] == nil {
				record[v] = " "
			}

			if arr_max_length[j] > 0 {
				arr_max_length[j] = int(math.Max(float64(arr_max_length[j]), float64(len(record[v].(string))+2)))
			} else {
				arr_max_length[j] = len(record[v].(string)) + 2
			}

			records[i] = append(records[i], record[v].(string))
		}
	}

	fmt.Println("Explain result:")
	//draw title
	this.drawLine(arr_max_length)
	this.drawData(fields, arr_max_length)
	//draw data
	for i, _ := range this.result {
		this.drawLine(arr_max_length)
		this.drawData(records[i], arr_max_length)
	}

	this.drawLine(arr_max_length)

	fmt.Println("")
} /*}}}*/

func (this *SqlExplain) drawLine(arr_length_list []int) { /*{{{*/
	fmt.Print("+")
	for _, length := range arr_length_list {
		fmt.Print(strings.Repeat("-", length), "+")
	}
	fmt.Println("")
} /*}}}*/

func (this *SqlExplain) drawData(arr_record_list []string, arr_length_list []int) { /*{{{*/
	fmt.Print("|")
	left := 0
	for i, value := range arr_record_list {
		space := int(math.Floor(float64(arr_length_list[i]-len(value)) / 2))
		left += space
		right := arr_length_list[i] - space
		format := "%" + strconv.Itoa(space) + "s%-" + strconv.Itoa(right) + "s|"

		fmt.Printf(format, " ", value)
		left -= space
		left += arr_length_list[i]
	}
	fmt.Println("")
} /*}}}*/
