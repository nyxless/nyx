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

func (s *SqlExplain) DrawConsole() { /*{{{*/
	arr_max_length := []int{}
	records := map[int][]string{}
	fields := def_fields

	if s.dbType != "mysql" {
		for _, record := range s.result {
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

	for i, record := range s.result {
		records[i] = []string{}

		for j, v := range fields {
			var val string
			if record[v] == nil {
				val = " "
			} else {
				val = fmt.Sprintf("%v", record[v])
			}

			if arr_max_length[j] > 0 {
				arr_max_length[j] = int(math.Max(float64(arr_max_length[j]), float64(len(val)+2)))
			} else {
				arr_max_length[j] = len(val) + 2
			}

			records[i] = append(records[i], val)
		}
	}

	fmt.Println("Explain result:")
	//draw title
	s.drawLine(arr_max_length)
	s.drawData(fields, arr_max_length)
	//draw data
	for i, _ := range s.result {
		s.drawLine(arr_max_length)
		s.drawData(records[i], arr_max_length)
	}

	s.drawLine(arr_max_length)

	fmt.Println("")
} /*}}}*/

func (s *SqlExplain) drawLine(arr_length_list []int) { /*{{{*/
	fmt.Print("+")
	for _, length := range arr_length_list {
		fmt.Print(strings.Repeat("-", length), "+")
	}
	fmt.Println("")
} /*}}}*/

func (s *SqlExplain) drawData(arr_record_list []string, arr_length_list []int) { /*{{{*/
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
