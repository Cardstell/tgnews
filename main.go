package main

import (
	"fmt"
	"os"
	"path/filepath"
	"encoding/json"
	"time"
	// "sort"
	"math/rand"
	"io/ioutil"
	"strings"
)

var (
	workers = 16
	ru_dbscan_minpts = 3
	ru_dbscan_eps = 0.011
	en_dbscan_minpts = 3
	en_dbscan_eps = 0.006
	timeWeightX = 1.0 / 86400
	timeWeightY = 0.005

	dim = 50
	// cat_en_model_file = "models/cat_en_v2-128.ftz"
	// cat_ru_model_file = "models/cat_ru.ftz"
	cat_en_model_file = "models/cat_en.ftz"
	cat_ru_model_file = "models/cat_ru.ftz"
	threads_en_model_file = "models/embeddings_en.bin"
	threads_ru_model_file = "models/embeddings_ru.bin"
)

type Article struct {
	Url, Site_name, Title, Description, Content string
	Published_time time.Time
	Filename, Path string
}

type LanguageItem struct {
	Lang_code string `json:"lang_code"`
	Articles []string `json:"articles"`
}

type CategoryItem struct {
	Category string `json:"category"`
	Articles []string `json:"articles"`
}

type ThreadItem struct {
	Title string `json:"title"`
	Articles []string `json:"articles"`
	Rank float64 `json:"-"`
	Category string `json:"category,omitempty"`
}

func replaceTags(text string) string {
	result := ""
	last := 0
	for i := 0;; {
		left := strings.Index(text[i:], "<")
		if left == -1 {
			break
		}
		left += i
		right := strings.Index(text[left:], ">")
		if right == -1 {
			break
		}
		right += left + 1
		i = right
		result += text[last:left]
		last = right
	}
	result += text[last:]

	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&ndash;", " ")
	result = strings.ReplaceAll(result, "&raquo;", " ")
	result = strings.ReplaceAll(result, "&laquo;", " ")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&#9;", " ")
	return result
}

func parseArticle(html string) (Article, error) {
	result := Article{}

	url_meta_prefix := "<meta property=\"og:url\" content=\""
	left := strings.Index(html, url_meta_prefix)
	if left > -1 {
		left += len(url_meta_prefix)
		right := strings.Index(html[left:], "\"/>")
		if right > -1 {
			right += left
			result.Url = replaceTags(html[left:right])
		}
	}

	site_name_meta_prefix := "<meta property=\"og:site_name\" content=\""
	left = strings.Index(html, site_name_meta_prefix)
	if left > -1 {
		left += len(site_name_meta_prefix)
		right := strings.Index(html[left:], "\"/>")
		if right > -1 {
			right += left
			result.Site_name = replaceTags(html[left:right])
		}
	}

	title_meta_prefix := "<meta property=\"og:title\" content=\""
	left = strings.Index(html, title_meta_prefix)
	if left > -1 {
		left += len(title_meta_prefix)
		right := strings.Index(html[left:], "\"/>")
		if right > -1 {
			right += left
			result.Title = replaceTags(html[left:right])
		}
	}

	description_meta_prefix := "<meta property=\"og:description\" content=\""
	left = strings.Index(html, description_meta_prefix)
	if left > -1 {
		left += len(description_meta_prefix)
		right := strings.Index(html[left:], "\"/>")
		if right > -1 {
			right += left
			result.Description = replaceTags(html[left:right])
		}
	}

	result.Published_time = time.Now()
	layout := "2006-01-02T15:04:05-07:00"
	time_meta_prefix := "<meta property=\"article:published_time\" content=\""
	left = strings.Index(html, time_meta_prefix)
	if left > -1 {
		left += len(time_meta_prefix)
		right := strings.Index(html[left:], "\"/>")
		if right > -1 {
			right += left
			var err error
			if result.Published_time, err = time.Parse(layout, replaceTags(html[left:right])); err != nil {
				return Article{}, err
			}
		}
	}

	for i := 0;; {
		left := strings.Index(html[i:], "<p>")
		if left == -1 {
			break
		}
		left += i + 3
		right := strings.Index(html[left:], "</p>")
		if right == -1 {
			break
		}
		right += left
		result.Content += replaceTags(html[left:right]) + " " 
		i = right
	}

	return result, nil
}

func loadArticles(path string) []Article {
	var result []Article

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(info.Name(), ".html") {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			b, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}

			article, err := parseArticle(string(b))
			if err != nil {
				panic(err)
			}
			article.Filename = info.Name()
			article.Path = path
			result = append(result, article)
			file.Close()
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return result
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println("Usage: ...")

	} else if args[0] == "languages" {
		loadLanguageModel()

		source_dir := args[1]
		articles := loadArticles(source_dir)

		langs := getLanguages(articles)
		lang_articles := map[string][]string{}
		for i := range articles {
			lang_articles[langs[i]] = append(lang_articles[langs[i]], articles[i].Filename)
		}
		result := []LanguageItem{}
		for lang, list := range lang_articles {
			result = append(result, LanguageItem{lang, list})
		}

		json_result, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(json_result))

	} else if args[0] == "news" {
		loadLanguageModel()
		loadCategoryModels()

		source_dir := args[1]
		articles := loadArticles(source_dir)

		categories := getCategories(articles)
		cat_articles := map[string][]string{}
		for i := range articles {
			cat_articles[categories[i]] = append(cat_articles[categories[i]], articles[i].Filename)
		}

		all_categories := []string{"society", "sports", "science", "technology", "entertainment", "economy", "other"}

		result := map[string][]string{}
		result["articles"] = []string{}
		for _, category := range all_categories {
			for _, article := range cat_articles[category] {
				result["articles"] = append(result["articles"], article)
			}
		}

		json_result, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(json_result))

	} else if args[0] == "categories" {
		loadLanguageModel()
		loadCategoryModels()

		source_dir := args[1]
		articles := loadArticles(source_dir)

		categories := getCategories(articles)
		cat_articles := map[string][]string{}
		for i := range articles {
			cat_articles[categories[i]] = append(cat_articles[categories[i]], articles[i].Filename)
		}

		all_categories := []string{"society", "sports", "science", "technology", "entertainment", "economy", "other"}

		result := []CategoryItem{}
		for _, category := range all_categories {
			result = append(result, CategoryItem{category, cat_articles[category]})
		}

		json_result, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(json_result))

	} else if args[0] == "threads" {
		loadLanguageModel()
		loadCategoryModels()
		loadThreadModels()

		source_dir := args[1]
		articles := loadArticles(source_dir)

		threads := getThreads(articles)
		json_result, err := json.Marshal(threads)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(json_result))

	} else if args[0] == "server" {
		loadLanguageModel()
		loadCategoryModels()
		loadThreadModels()

		startServer(args[1])
	
	} else if args[0] == "parse" {
		source_dir := args[1]
		articles := loadArticles(source_dir)

		output, err := json.Marshal(articles)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(output))

	} else {
		fmt.Println("Usage: ...")
	}
}