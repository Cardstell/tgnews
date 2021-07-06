package main

import (
	// "os"
	"strings"
	// "io/ioutil"
	"bytes"
)

var (
	// ru_embeddings_file = "models/skipgram_ru_256"
	// en_embeddings_file = "models/skipgram_en_256"
	// ru_dict = map[string]int{}
	// en_dict = map[string]int{}

	// max_len = 256
)

func preprocessText(text string) string {
	pattern1 := "-’0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZабвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ"
	pattern2 := ",.!?"
	var buffer bytes.Buffer
	for _, char := range text {
		inPattern1 := false
		for _, ch := range pattern1 {
			if ch == char {
				inPattern1 = true
				break
			}
		}
		inPattern2 := false
		for _, ch := range pattern2 {
			if ch == char {
				inPattern2 = true
				break
			}
		}
		if inPattern1 {
			buffer.WriteString(string(char))
		} else if inPattern2 {
			buffer.WriteString(" " + string(char) + " ")
		} else {
			buffer.WriteString(" ")
		}
	}
	text = strings.ToLower(buffer.String())
	return strings.Join(strings.Fields(text), " ")
}

func preprocessText2(text string) string {
	pattern1 := "-’0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZабвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ"
	pattern2 := ",.!?"
	var buffer bytes.Buffer
	for _, char := range text {
		inPattern1 := false
		for _, ch := range pattern1 {
			if ch == char {
				inPattern1 = true
				break
			}
		}
		inPattern2 := false
		for _, ch := range pattern2 {
			if ch == char {
				inPattern2 = true
				break
			}
		}
		if inPattern1 {
			buffer.WriteString(string(char))
		} else if inPattern2 {
			buffer.WriteString(" " + string(char) + " ")
		} else {
			buffer.WriteString(" ")
		}
	}
	text = strings.ToLower(buffer.String())
	fields := strings.Fields(text)
	fields2 := []string{}
	for i := range fields {
		if len(fields2) > 300 {
			break
		}
		if !stop_dict[fields[i]] {
			fields2 = append(fields2, fields[i])
		}
	}
	return strings.Join(fields2, " ")
}

// func loadEnEmbeddings() {
// 	file, err := os.Open(en_embeddings_file + ".vocab")
// 	if err != nil {
// 		panic(err)
// 	}
// 	b, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		panic(err)
// 	}
// 	vocab := strings.Split(string(b), "\n")
// 	for i := 0;i<len(vocab);i++ {
// 		en_dict[vocab[i]] = i + 1
// 	}
// }

// func loadRuEmbeddings() {
// 	file, err := os.Open(ru_embeddings_file + ".vocab")
// 	if err != nil {
// 		panic(err)
// 	}
// 	b, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		panic(err)
// 	}
// 	vocab := strings.Split(string(b), "\n")
// 	for i := 0;i<len(vocab);i++ {
// 		ru_dict[vocab[i]] = i + 1
// 	}
// }

// func tokenizeTextEn(text string) []int32 {
// 	words := strings.Split(text, " ")
// 	result := make([]int32, max_len)
// 	for i := range result {
// 		result[i] = 0
// 	}
// 	cnt := 0
// 	for i := range words {
// 		index, ok := en_dict[words[i]]
// 		if !ok {
// 			continue
// 		}
// 		result[cnt] = int32(index)
// 		cnt += 1
// 		if cnt == max_len {
// 			break
// 		}
// 	}
// 	return result
// }

// func tokenizeTextRu(text string) []int32 {
// 	words := strings.Split(text, " ")
// 	result := make([]int32, max_len)
// 	for i := range result {
// 		result[i] = 0
// 	}
// 	cnt := 0
// 	for i := range words {
// 		index, ok := ru_dict[words[i]]
// 		if !ok {
// 			continue
// 		}
// 		result[cnt] = int32(index)
// 		cnt += 1
// 		if cnt == max_len {
// 			break
// 		}
// 	}
// 	return result
// }