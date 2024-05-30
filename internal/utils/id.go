package utils

import (
	"strconv"
)

func ParseID(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

func FormatID(value int64) string {
	return strconv.FormatInt(value, 10)
}

func ParseIDList(values []string) ([]int64, error) {
	result := make([]int64, 0, len(values))

	for _, value := range values {
		id, err := ParseID(value)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}

	return result, nil
}

func FormatIDList(values []int64) []string {
	result := make([]string, 0, len(values))

	for _, value := range values {
		result = append(result, FormatID(value))
	}

	return result
}
