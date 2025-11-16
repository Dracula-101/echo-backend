package utils

import "fmt"

func IntToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func StringToInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func StringToMustInt(s string) int {
	i, err := StringToInt(s)
	if err != nil {
		panic(fmt.Sprintf("failed to convert string to int: %v", err))
	}
	return i
}
