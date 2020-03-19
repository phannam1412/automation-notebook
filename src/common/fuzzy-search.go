package common

import (
	"sort"
	"strings"
)

type FuzzySearch struct {
	Lines []string
}

func NewFuzzySearch(input []string) *FuzzySearch {
	for k, v := range input {
		input[k] = strings.ToLower(v)
	}
	return &FuzzySearch{Lines: input}
}

func (this *FuzzySearch) Find(input string, limit int) []string {
	input = strings.ToLower(input)
	keywords := strings.Split(input, " ")
	var res []string
	for _, v := range keywords {
		if len(strings.Trim(v, " ")) > 0 {
			res = append(res, v)
		}
	}
	var output []string
	for k, command := range this.Lines {
		pos := 0
		command = strings.ToLower(command)
		for _, userKeywordToken := range res {
			pos = strings.Index(command, userKeywordToken)
			if pos == -1 {
				break
			}
			command = command[(pos + len(userKeywordToken)):]
		}
		if pos != -1 {
			output = append(output, this.Lines[k])
			if len(output) >= limit {
				break
			}
		}
	}
	sort.Slice(output, func(i, j int) bool {
		return len(output[j]) > len(output[i])
	})
	return output
}
