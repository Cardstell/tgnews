package main

import (
	"sync"
	"math"
	"os"
	"io/ioutil"
	// "fmt"
	"strings"
	// "math/rand"
	"strconv"
)

var (
	stop_dict = map[string]bool{}
)

type VectorClusterable struct {
	coords []float32
	publishedTime int64
	num int
}

type ChItemVector struct {
	Article *Article
	Result *[]float32
}

func normalize(x []float32) []float32 {
	k := 0.0

	for i := range x {
		k += float64(x[i]) * float64(x[i])
	}
	k = 1.0 / math.Sqrt(k)

	for i := range x {
		x[i] *= float32(k)
	}
	return x
}

func loadStopWords() {
	f, err := os.Open("models/stop_word_english.txt")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	words := strings.Split(string(b), "\n")
	for _, word := range words {
		stop_dict[word] = true
	}
	f.Close()

	f, err = os.Open("models/stop_word_russian.txt")
	if err != nil {
		panic(err)
	}
	b, err = ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	words = strings.Split(string(b), "\n")
	for _, word := range words {
		stop_dict[word] = true
	}
}

func loadThreadModels() {
	loadStopWords()
	LoadModel("threads_en", threads_en_model_file)
	LoadModel("threads_ru", threads_ru_model_file)
}

func getVectorEn(article *Article) []float32 {
	text := article.Title + " " + article.Description + " " + article.Content
	// text := article.Title
	return normalize(SentenceVector("threads_en", preprocessText2(text)))
}

func getVectorRu(article *Article) []float32 {
	text := article.Title + " " + article.Description + " " + article.Content
	// text := article.Title
	return normalize(SentenceVector("threads_ru", preprocessText2(text)))
}

func workerThread(in chan ChItemVector, wg *sync.WaitGroup, lang string) {
	for {
		item, ok := <-in
		if !ok {
			break
		}
		if lang == "en" {
			*(item.Result) = getVectorEn(item.Article)
		} else if lang == "ru" {
			*(item.Result) = getVectorRu(item.Article)
		}
	}
	wg.Done()
}

func CosineDistance(a, b []float32) float64 {
	result := float64(0)
	for i := range a {
		result += float64(a[i]) * float64(b[i])
	}

	// lengthA := float64(0)
	// for i := range a {
	// 	lengthA += float64(a[i]) * float64(a[i])
	// }
	// lengthA = math.Sqrt(lengthA)

	// lengthB := float64(0)
	// for i := range b {
	// 	lengthB += float64(b[i]) * float64(b[i])
	// }
	// lengthB = math.Sqrt(lengthB)

	// return 0.5 * (1.0 - result / (lengthA * lengthB))

	return 0.5 * (1.0 - result)
}

func EuclideanDistance(a, b []float32) float64 {
	result := float64(0)
	for i := range a {
		result += float64(a[i] - b[i]) * float64(a[i] - b[i])
	}
	return math.Sqrt(result)
}

func Distance(a, b []float32) float64 {
	// return EuclideanDistance(a, b)
	return CosineDistance(a, b)
}

func (v VectorClusterable) Distance(c interface{}) float64 {
	distance := Distance(v.coords, c.(VectorClusterable).coords)
	diff := timeWeightX * float64(v.publishedTime - c.(VectorClusterable).publishedTime)
	distance += timeWeightY * (1.0 - math.Exp(-diff*diff))
	return distance
}

func (v VectorClusterable) GetID() string {
	return strconv.Itoa(v.num)
}

func getTitle(titles []string, vectors [][]float32) string {
	vector := make([]float32, len(vectors[0]))

	for i := range vectors {
		for j := range vectors[i] {
			vector[j] += vectors[i][j] / float32(len(vectors))
		}
	}

	minDist := 10000.0
	index := 0
	for i := range vectors {
		dist := Distance(vector, vectors[i])
		if dist < minDist {
			minDist = dist
			index = i
		}
	}
	return titles[index]
}

func getThreadsInGroup(articles []*Article, lang string) []ThreadItem {
	vectors := make([][]float32, len(articles))
	var inputs []chan ChItemVector
	var wg sync.WaitGroup

	for i := 0;i<workers;i++ {
		wg.Add(1)
		inputs = append(inputs, make(chan ChItemVector, len(articles)))
		go workerThread(inputs[len(inputs)-1], &wg, lang)
	}	

	for i := range articles {
		inputs[i%workers] <- ChItemVector{articles[i], &vectors[i]}
	}

	for i := 0;i<workers;i++ {
		close(inputs[i])
	}
	wg.Wait()
	
	clusterList := make([]Clusterable, len(articles))
	for i := range articles {
		clusterList[i] = VectorClusterable{vectors[i], articles[i].Published_time.Unix(), i}
	}

	minpts := en_dbscan_minpts
	eps := en_dbscan_eps

	if lang == "ru" {
		minpts = ru_dbscan_minpts
		eps = ru_dbscan_eps
	}

	clusters := Clusterize(clusterList, minpts, eps)
	not_used := make([]bool, len(articles))
	for i := range articles {
		not_used[i] = true
	}

	result := []ThreadItem{}
	for _, cluster := range clusters {
		th := ThreadItem{}
		threadVectors := make([][]float32, len(cluster))
		titles := make([]string, len(cluster))

		for j := range cluster {
			index := cluster[j].(VectorClusterable).num
			not_used[index] = false
			th.Articles = append(th.Articles, articles[index].Filename)
			threadVectors[j] = vectors[index]
			titles[j] = articles[index].Title
		}

		th.Title = getTitle(titles, threadVectors)
		result = append(result, th)
	}
	for i := range articles {
		if not_used[i] {
			result = append(result, ThreadItem{articles[i].Title, 
				[]string{articles[i].Filename}, 0, ""})
		}
	}

	return result
}

func getThreads(articles []Article) []ThreadItem {
	langs := getLanguages(articles)
	categories := getCategories(articles)

	groups := map[string][]*Article{}

	for i := range articles {
		if (langs[i] == "ru" || langs[i] == "en") && categories[i] != "not_news" {
			groups[langs[i] + "_" + categories[i]] = append(groups[langs[i] + "_" + categories[i]], 
				&articles[i])
		}
	}

	result := []ThreadItem{}

	for group_name, group_articles := range groups {
		result = append(result, getThreadsInGroup(group_articles, group_name[:2])...)
	}
	return result
}