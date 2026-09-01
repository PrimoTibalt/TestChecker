package main

import (
	"strings"
	"unicode/utf8"
)

// IgnoreIndentation drops the whitespace that only lays a snippet out — the
// spaces and tabs around every line, and blank lines altogether — so that a
// code answer typed at a different indentation still comes out at distance 0.
func IgnoreIndentation(text string) string {
	lines := []string{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.Trim(line, " \t\r")
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func Ld(actual, expected string) (distance int) {
	m, n := utf8.RuneCountInString(expected)+1, utf8.RuneCountInString(actual)+1

	arr := make([][]int, m)
	for i := range arr {
		arr[i] = make([]int, n)
	}

	for i := range arr {
		for j := range arr[i] {
			distance := calcNextCell(i, j, arr, actual, expected)
			arr[i][j] = distance
		}
	}

	return arr[m-1][n-1]
}

func calcNextCell(i, j int, arr [][]int, actual, expected string) int {
	if i == 0 && j == 0 {
		return 0
	}

	if i == 0 {
		return arr[i][j-1] + 1
	}

	if j == 0 {
		return arr[i-1][j] + 1
	}

	eR := []rune(expected)[i-1]
	aR := []rune(actual)[j-1]
	if eR == aR {
		return arr[i-1][j-1]
	}

	return min(arr[i][j-1], arr[i-1][j-1], arr[i-1][j]) + 1
}
