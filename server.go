package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"math"
	"sort"
	"sync"
	"math/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"time"
	"errors"
	"strings"
	"github.com/gorilla/mux"
	"github.com/boltdb/bolt"
	
)

var (
	clusterIDlength = 12
	categoriesList = []string{"society", "sports", "science", "technology", "entertainment", "economy", "other", "not_news"}
	db *bolt.DB
	globalWaitGroup sync.WaitGroup
)

func categoryIndex(category string) int {
	for i := range categoriesList {
		if category == categoriesList[i] { 
			return i
		}
	}
	return -1
}

func randomClusterID() string {
	id := make([]byte, clusterIDlength)
	rand.Read(id)
	return hex.EncodeToString(id)
}

func zeroClusterID() string {
	return hex.EncodeToString(make([]byte, clusterIDlength))
}

func randomVector() []float32 {
	vector := make([]float32, dim)
	for i := range vector {
		vector[i] = rand.Float32()
	}
	return vector
}

func timeToBytes(t time.Time) []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, uint64(t.Unix()))
	return result
}

func encodeArticle(lang, category string, publishedTime, endTime time.Time, neighbours int, clusterID string, vector []float32) ([]byte, error) {
	if category == "not_news" || (lang != "ru" && lang != "en") {
		return []byte{0}, nil
	}
	result := make([]byte, 22 + clusterIDlength + dim * 4)
	if lang == "en" {
		result[0] = byte(0)
	} else if lang == "ru" {
		result[0] = byte(1)
	}
	result[1] = byte(categoryIndex(category))

	binary.BigEndian.PutUint64(result[2:10], uint64(publishedTime.Unix()))
	binary.BigEndian.PutUint64(result[10:18], uint64(endTime.Unix()))
	binary.BigEndian.PutUint32(result[18:22], uint32(neighbours))
	
	var err error
	decoded, err := hex.DecodeString(clusterID) 
	if err != nil {
		return nil, err
	}
	for i := 0; i < clusterIDlength; i++ {
		result[22 + i] = decoded[i]
	}

	for i := range vector {
		offset := 22 + clusterIDlength + i*4
		binary.BigEndian.PutUint32(result[offset:offset+4], math.Float32bits(vector[i]))
	}

	return result, nil
}

func decodeArticle(b []byte) (lang, category string, publishedTime, endTime time.Time, neighbours int, clusterID string, vector []float32, err error) {
	if len(b) == 1 {
		return "other", "not_news", time.Time{}, time.Time{}, 0, "", nil, nil
	} else if len(b) == 22 + clusterIDlength + dim * 4 {
		if b[0] == byte(1) {
			lang = "ru"
		} else {
			lang = "en"
		}

		if b[1] < 0 || int(b[1]) >= len(categoriesList) {
			return "", "", time.Time{}, time.Time{}, 0, "", nil, errors.New("invalid category byte")
		}
		category = categoriesList[b[1]]

		publishedTime = time.Unix(int64(binary.BigEndian.Uint64(b[2:10])), 0)
		endTime = time.Unix(int64(binary.BigEndian.Uint64(b[10:18])), 0)
		neighbours = int(binary.BigEndian.Uint32(b[18:22]))

		clusterID = hex.EncodeToString(b[22:22+clusterIDlength])

		vector = make([]float32, dim)
		for i := range vector {
			offset := 22 + clusterIDlength + i*4
			vector[i] = math.Float32frombits(binary.BigEndian.Uint32(b[offset:offset+4]))
		}
		return
	} else {
		return "", "", time.Time{}, time.Time{}, 0, "", nil, errors.New("invalid length")
	}
}

func getNeighboursForFirstPoint(bucket *bolt.Bucket, name, lang, category string, vector []float32) ([]string, []string, []string, error) {
	noise := []string{}
	changeState := []string{}
	clustered := []string{}

	minpts := en_dbscan_minpts
	eps := en_dbscan_eps

	if lang == "ru" {
		minpts = ru_dbscan_minpts
		eps = ru_dbscan_eps
	}

	cursor := bucket.Cursor()
	for name_article, article := cursor.First(); name_article != nil; name_article, article = cursor.Next() {
		lang_, category_, _, _, neighbours, _, vector_, err := decodeArticle(article)
		if err != nil {
			return nil, nil, nil, err
		}
		if lang != lang_ || category != category_ || name == string(name_article) {
			continue
		}

		if Distance(vector, vector_) < eps {
			if neighbours + 2 < minpts {
				noise = append(noise, string(name_article))
			} else if neighbours + 2 == minpts {
				changeState = append(changeState, string(name_article))
			} else {
				clustered = append(clustered, string(name_article))
			}
		}
	}

	return noise, changeState, clustered, nil
}

func getNeighbours(bucket *bolt.Bucket, name, lang, category string) ([]string, error) {
	result := []string{}
	_, _, _, _, _, _, vector, err := decodeArticle(bucket.Get([]byte(name)))
	if err != nil {
		return nil, err
	}

	eps := en_dbscan_eps

	if lang == "ru" {
		eps = ru_dbscan_eps
	}

	cursor := bucket.Cursor()
	for name_article, article := cursor.First(); name_article != nil; name_article, article = cursor.Next() {
		lang_, category_, _, _, _, _, vector_, err := decodeArticle(article)
		if err != nil {
			return nil, err
		}
		if lang_ != lang || category != category_ || string(name_article) == name {
			continue
		}

		if Distance(vector, vector_) < eps {
			result = append(result, string(name_article))
		}
	}

	return result, nil
}

func getCoreNeighbours(bucket *bolt.Bucket, name, name2, lang, category string) ([]string, error) {
	clustered := []string{}
	_, _, _, _, _, clusterID, vector, err := decodeArticle(bucket.Get([]byte(name)))
	if err != nil {
		return nil, err
	}

	minpts := en_dbscan_minpts
	eps := en_dbscan_eps

	if lang == "ru" {
		minpts = ru_dbscan_minpts
		eps = ru_dbscan_eps
	}

	cursor := bucket.Cursor()
	for name_article, article := cursor.First(); name_article != nil; name_article, article = cursor.Next() {
		lang_, category_, _, _, neighbours, clusterID_, vector_, err := decodeArticle(article)
		if err != nil {
			return nil, err
		}
		if lang_ != lang || category != category_ || string(name_article) == name || string(name_article) == name2 || clusterID == clusterID_ {
			continue
		}

		if Distance(vector, vector_) < eps {
			if neighbours + 1 >= minpts {
				clustered = append(clustered, string(name_article))
			}
		}
	}

	return clustered, nil
}

func mergeClusters(tx *bolt.Tx, lang, category string, names []string) (string, error) {
	if len(names) <= 1 {
		return zeroClusterID(), nil
	}
	articles_bucket := tx.Bucket([]byte("articles"))
	clusters_bucket := tx.Bucket([]byte(lang + "_" + category))

	articles := map[string]bool{}
	clusters := map[string]bool{}

	for i := range names {
		articles[names[i]] = true
		_, _, _, _, _, clusterID, _, err := decodeArticle(articles_bucket.Get([]byte(names[i])))
		if err != nil {
			return "", err
		}

		if clusterID != zeroClusterID() {
			clusters[clusterID] = true
		}	
	}

	for clusterID, _ := range clusters {
		var clusterArticles []string
		err := json.Unmarshal(clusters_bucket.Get([]byte(clusterID)), &clusterArticles)
		if err != nil {
			return "", err
		}
		for _, article := range clusterArticles {
			articles[article] = true
		}
		clusters_bucket.Delete([]byte(clusterID))
	}

	clusterID := randomClusterID()
	clusterArticles := []string{}

	for article, _ := range articles {
		clusterArticles = append(clusterArticles, article)

		_, _, t1, t2, neighbours, _, vector, err := decodeArticle(articles_bucket.Get([]byte(article)))
		if err != nil {
			return "", err
		}
		encoded, err := encodeArticle(lang, category, t1, t2, neighbours, clusterID, vector)
		if err != nil {
			return "", err
		}
		articles_bucket.Put([]byte(article), encoded)
	}

	jsonClusterArticles, err := json.Marshal(clusterArticles)
	if err != nil {
		return "", err
	}
	clusters_bucket.Put([]byte(clusterID), []byte(jsonClusterArticles))
	return clusterID, nil
}

func addArticle(name, title, lang, category string, publishedTime, endTime time.Time, vector []float32) int {
	articleExists := false
	err := db.Update(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte(name)) != nil {
			articleExists = true
			return nil
		}

		noiseNeigh, changeStateNeigh, clusteredNeigh, err := getNeighboursForFirstPoint(
			articles_bucket, name, lang, category, vector)
		if err != nil {
			return err
		}

		minpts := en_dbscan_minpts

		if lang == "ru" {
			minpts = ru_dbscan_minpts
		}

		clustered := len(noiseNeigh) + len(changeStateNeigh) + len(clusteredNeigh) + 1 >= minpts

		for i := range changeStateNeigh {
			coreNeigh, err := getCoreNeighbours(articles_bucket, changeStateNeigh[i], name, lang, category)
			if err != nil {
				return err
			}

			_, err = mergeClusters(tx, lang, category, append(coreNeigh, changeStateNeigh[i]))
			if err != nil {
				return err
			}
		}

		var clusterID string
		if clustered {
			clusterID, err = mergeClusters(tx, lang, category, append(changeStateNeigh, clusteredNeigh...))
			if err != nil {
				return err
			}
		} else {
			clusterID = zeroClusterID()
			if len(clusteredNeigh) != 0 {
				_, _, _, _, _, clusterID, _, err = decodeArticle(articles_bucket.Get([]byte(clusteredNeigh[0])))
				if err != nil {
					return err
				}
			} else if len(changeStateNeigh) != 0 {
				_, _, _, _, _, clusterID, _, err = decodeArticle(articles_bucket.Get([]byte(changeStateNeigh[0])))
				if err != nil {
					return err
				}
			}
		}

		all_neighbours := append(noiseNeigh, append(changeStateNeigh, clusteredNeigh...)...)

		for _, article_name := range all_neighbours {
			_, _, t1, t2, neighbours, clusterID_, vector_, err := decodeArticle(articles_bucket.Get([]byte(article_name)))
			if err != nil {
				return err
			}
			encoded, err := encodeArticle(lang, category, t1, t2, neighbours + 1, clusterID_, vector_)
			if err != nil {
				return err
			}
			articles_bucket.Put([]byte(article_name), encoded)
		}

		encoded, err := encodeArticle(lang, category, publishedTime, endTime, len(noiseNeigh) + len(changeStateNeigh) + len(clusteredNeigh), 
			clusterID, vector)
		if err != nil {
			return err
		}
		articles_bucket.Put([]byte(name), encoded)

		titles_bucket := tx.Bucket([]byte("titles"))
		titles_bucket.Put([]byte(name), []byte(title))

		times_bucket := tx.Bucket([]byte("times"))
		times_bucket.Put(timeToBytes(endTime), []byte(name))
		start_times_bucket := tx.Bucket([]byte("starts"))
		start_times_bucket.Put(timeToBytes(publishedTime), []byte(name))


		if clusterID != zeroClusterID() {
			clusters_bucket := tx.Bucket([]byte(lang + "_" + category))
			clusterArticles := []string{}
			if clusters_bucket.Get([]byte(clusterID)) != nil {
				err := json.Unmarshal(clusters_bucket.Get([]byte(clusterID)), &clusterArticles)
				if err != nil {
					return err
				}
			}


			clusterArticles = append(clusterArticles, name)

			jsonClusterArticles, err := json.Marshal(clusterArticles)
			if err != nil {
				return err
			}

			clusters_bucket.Put([]byte(clusterID), jsonClusterArticles)
		}

		return nil
	})
	if err != nil {
		return 500
	}

	if articleExists {
		deleteArticle(name)
		if addArticle(name, title, lang, category, publishedTime, endTime, vector) == 201 {
			return 204
		} else {
			return 500
		}		
	}
	return 201
}

func deleteArticle(name string) int {
	code := 204
	err := db.Update(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte(name)) == nil {
			code = 404
			return nil
		}

		lang, category, publishedTime, endTime, _, clusterID, _, err := decodeArticle(
			articles_bucket.Get([]byte(name)))
		if err != nil {
			return err
		}

		if category != "not_news" {
			if clusterID != zeroClusterID() {
				clusters_bucket := tx.Bucket([]byte(lang + "_" + category))
				clusterArticles := []string{}
				err = json.Unmarshal(clusters_bucket.Get([]byte(clusterID)), &clusterArticles)
				if err != nil {
					return err
				}

				newClusterArticles := []string{}
				for _, name_ := range clusterArticles {
					if name_ != name {
						newClusterArticles = append(newClusterArticles, name_)
					}
				}

				if len(newClusterArticles) == 0 {
					clusters_bucket.Delete([]byte(clusterID))
				} else {
					jsonClusterArticles, err := json.Marshal(newClusterArticles)
					if err != nil {
						return err
					}
					clusters_bucket.Put([]byte(clusterID), jsonClusterArticles)
				}
			}

			all_neighbours, err := getNeighbours(articles_bucket, name, lang, category)
			if err != nil {
				return err
			}

			for _, article_name := range all_neighbours {
				_, _, t1, t2, neighbours, clusterID_, vector_, err := decodeArticle(articles_bucket.Get([]byte(article_name)))
				if err != nil {
					return err
				}
				encoded, err := encodeArticle(lang, category, t1, t2, neighbours - 1, clusterID_, vector_)
				if err != nil {
					return err
				}
				articles_bucket.Put([]byte(article_name), encoded)
			}

			titles_bucket := tx.Bucket([]byte("titles"))
			titles_bucket.Delete([]byte(name))
		}

		articles_bucket.Delete([]byte(name))
		times_bucket := tx.Bucket([]byte("times"))
		times_bucket.Delete([]byte(timeToBytes(endTime)))
		start_times_bucket := tx.Bucket([]byte("starts"))
		start_times_bucket.Delete([]byte(timeToBytes(publishedTime)))
		return nil
	})
	if err != nil {
		return 500
	}

	return code
}

func deleteOldArticles() error {
	oldArticles := []string{}

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("times"))
		cursor := bucket.Cursor()
		for t, name := cursor.First(); t != nil; t, name = cursor.Next() {
			if binary.BigEndian.Uint64(t) < uint64(time.Now().Unix())  {
				oldArticles = append(oldArticles, string(name))
			} else {
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return err
	}


	for i := range oldArticles {
		if deleteArticle(oldArticles[i]) == 500 {
			return errors.New("error deleting article")
		}
	}

	return nil
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	go deleteOldArticles();

	name := mux.Vars(r)["name"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if len(r.Header["Cache-Control"]) != 1 {
		http.Error(w, "Bad Request", 400)
		return
	}
	fields := strings.Split(r.Header["Cache-Control"][0], "=")
	if len(fields) != 2 {
		http.Error(w, "Bad Request", 400)
		return
	}

	ttl, err := strconv.Atoi(fields[1])
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	article, err := parseArticle(string(body))
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return		
	}

	lang := getLanguage(&article)
	var category string
	notnews := false

	if lang == "en" {
		category = getCategoryEn(&article)
	} else if lang == "ru" {
		category = getCategoryRu(&article)
	} else {
		notnews = true
	}

	if category == "not_news" {
		notnews = true
	}

	endTime := article.Published_time.Add(time.Second * time.Duration(ttl))

	code := 201
	err = db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte(name)) != nil {
			code = 204
		} 
		return nil
	})
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if code == 201 {
		w.WriteHeader(201)
		w.Write([]byte("Created"))
	} else if code == 204 {
		w.WriteHeader(204)
		w.Write([]byte("Updated"))
	}

	if notnews {
		globalWaitGroup.Add(1)
		go func() {
				db.Update(func(tx *bolt.Tx) error {
				articles_bucket := tx.Bucket([]byte("articles"))
				times_bucket := tx.Bucket([]byte("times"))
				start_times_bucket := tx.Bucket([]byte("starts"))

				encoded, err := encodeArticle("", "not_news", time.Time{}, time.Time{}, 0, "", nil)
				if err != nil {
					return err
				}

				articles_bucket.Put([]byte(name), encoded)
				times_bucket.Put(timeToBytes(endTime), []byte(name))
				start_times_bucket.Put(timeToBytes(article.Published_time), []byte(name))
				return nil
			})
			globalWaitGroup.Done()
		}()

		return
	}

	globalWaitGroup.Add(1)
	go func() {
		var vector []float32
		if lang == "en" {
			vector = getVectorEn(&article)
		} else if lang == "ru" {
			vector = getVectorRu(&article)
		}

		addArticle(name, article.Title, lang, category, article.Published_time, endTime, vector)
		globalWaitGroup.Done()
	}()
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	go deleteOldArticles()

	name := mux.Vars(r)["name"]

	globalWaitGroup.Add(1)
	go func() {
		deleteArticle(name)
		globalWaitGroup.Done()
	}()
	code := 404
	err := db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte(name)) != nil {
			code = 204
		} 
		return nil
	})
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if code == 204 {
		w.WriteHeader(204)
		w.Write([]byte("No Content"))
	} else if code == 404 {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
	}
}

func getLastTime() int64 {
	var result int64
	db.View(func(tx *bolt.Tx) error {
		lastTime, _ := tx.Bucket([]byte("starts")).Cursor().Last()
		if lastTime == nil {
			result = 0
		} else {
			result = int64(binary.BigEndian.Uint64(lastTime))
		}
		return nil
	})
	return result
}

func getRank(clusterTime int64, clusterSize int) float64 {
	return float64(clusterSize)
}

func threadsHandler(w http.ResponseWriter, r *http.Request) {
	go deleteOldArticles()
	globalWaitGroup.Wait()
	lang := r.FormValue("lang_code")
	category_ := r.FormValue("category")
	period, err := strconv.Atoi(r.FormValue("period"))
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	endTime := getLastTime()
	startTime := endTime - int64(period)

	categories := []string{"society", "sports", "science", "technology", "entertainment", "economy", "other"}
	if category_ != "any" {
		categories = []string{category_}
	}

	threads := []ThreadItem{}

	for _, category := range categories {
		out_category := ""
		if category_ == "any" {
			out_category = category
		}
		err = db.View(func(tx *bolt.Tx) error {
			articles_bucket := tx.Bucket([]byte("articles"))
			titles_bucket := tx.Bucket([]byte("titles"))
			clusters_bucket := tx.Bucket([]byte(lang + "_" + category))

			c := clusters_bucket.Cursor()
			for key, cluster := c.First(); key != nil; key, cluster = c.Next() {
				currentThread := ThreadItem{}
				titles := []string{}
				vectors := [][]float32{}

				var articles []string
				err := json.Unmarshal(cluster, &articles)
				if err != nil {
					return err
				}

				times := []int64{}
				for _, article_name := range articles {
					_, _, publishedTime, _, _, _, vector, err := decodeArticle(articles_bucket.Get([]byte(article_name)))
					if err != nil {
						return err
					}
					if publishedTime.Unix() < startTime {
						continue
					}

					currentThread.Articles = append(currentThread.Articles, article_name)
					titles = append(titles, string(titles_bucket.Get([]byte(article_name))))
					vectors = append(vectors, vector)
					times = append(times, publishedTime.Unix())
				}

				if len(currentThread.Articles) == 0 {
					continue
				}
				sort.Slice(times, func(i, j int) bool {
					return times[i] < times[j]
				})
				clusterTime := times[int(0.8 * float64(len(times)))]
				currentThread.Title = getTitle(titles, vectors)
				currentThread.Rank = getRank(clusterTime, len(articles))
				currentThread.Category = out_category
				threads = append(threads, currentThread)
			}
			return nil
		})
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
	}
	err = db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		titles_bucket := tx.Bucket([]byte("titles"))

		c := articles_bucket.Cursor()
		for article_name, article := c.First(); article_name != nil; article_name, article = c.Next() {
			lang_, artCategory, t1, _, _, clusterID, _, err := decodeArticle(article)
			if err != nil {
				return err
			}
			if lang_ == lang && (artCategory == category_ || category_ == "any") && t1.Unix() >= startTime && 
				clusterID == zeroClusterID() {

				out_category := ""
				if category_ == "any" {
					out_category = artCategory
				}

				threads = append(threads, ThreadItem{string(titles_bucket.Get(article_name)), 
					[]string{string(article_name)}, getRank(t1.Unix(), 1), out_category})
			}
		}
		return nil

	}) 
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	sort.Slice(threads, func(i, j int) bool {
		return threads[i].Rank > threads[j].Rank
	})

	if len(threads) > 1000 {
		threads = threads[:1000]
	} 

	result := map[string][]ThreadItem{"threads": threads}
	json_result, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(json_result))
}

func startServer(port string) {
	var err error
	db, err = bolt.Open("database.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("articles"))
		tx.CreateBucketIfNotExists([]byte("titles"))
		tx.CreateBucketIfNotExists([]byte("times"))
		tx.CreateBucketIfNotExists([]byte("starts"))

		for _, lang := range []string{"en", "ru"} {
			for _, cat := range []string{"society", "sports", "science", "technology", "entertainment", "economy", "other"} {
				tx.CreateBucketIfNotExists([]byte(lang + "_" + cat))
			}
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{name}", putHandler).Methods("PUT")
	r.HandleFunc("/{name}", deleteHandler).Methods("DELETE")
	r.HandleFunc("/threads", threadsHandler).Methods("GET")
	if err = fasthttp.ListenAndServe(":" + port, fasthttpadaptor.NewFastHTTPHandler(r)); err != nil {
		panic(err)
	}

	// if err := fasthttp.ListenAndServe(":" + port, func(ctx *fasthttp.RequestCtx) {
	// 	fmt.Fprintln(ctx, ctx.RequestURI())
	// }); err != nil {
	// 	panic(err)
	// }
}