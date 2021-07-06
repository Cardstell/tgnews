package main

import (
	"sync"
)

func loadCategoryModels() {
	LoadModel("cat_ru", cat_ru_model_file)
	LoadModel("cat_en", cat_en_model_file)
}

func getCategoryEn(article *Article) string {
	text := article.Title + " " + article.Description + " " + article.Content
	result, err := Predict("cat_en", preprocessText(text), 1)
	if err != nil {
		panic(err)
	}

	for label, _ := range result {
		return label[9:]
	}
	return "not_news"
}

func getCategoryRu(article *Article) string {
	text := article.Title + " " + article.Description + " " + article.Content
	result, err := Predict("cat_ru", preprocessText(text), 1)
	if err != nil {
		panic(err)
	}

	for label, _ := range result {
		return label[9:]
	}
	return "not_news"
}

func getCategory(article *Article) string {
	lang := getLanguage(article)
	if lang == "en" {
		return getCategoryEn(article)
	} else if lang == "ru" {
		return getCategoryRu(article)
	} 
	return "not_news"
}

func workerCategories(in chan ChItem, wg *sync.WaitGroup) {
	for {
		item, ok := <-in
		if !ok {
			break
		}
		*(item.Result) = getCategory(item.Article)
	}
	wg.Done()
}

func getCategories(articles []Article) []string {
	result := make([]string, len(articles))
	var inputs []chan ChItem
	var wg sync.WaitGroup

	for i := 0;i<workers;i++ {
		wg.Add(1)
		inputs = append(inputs, make(chan ChItem, len(articles)))
		go workerCategories(inputs[len(inputs)-1], &wg)
	}

	for i := range articles {
		inputs[i%workers] <- ChItem{&articles[i], &result[i]}
	}

	for i := 0;i<workers;i++ {
		close(inputs[i])
	}
	wg.Wait()

	return result
}