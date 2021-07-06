package main

import (
	"math/rand"
	"testing"
	"time"
	"fmt"
	"encoding/json"
	"github.com/boltdb/bolt"
)

func TestEncodeDecode(t *testing.T) {
	N := 10000
	for i := 0; i < N; i++ {
		lang := []string{"ru", "en", "other"}[rand.Intn(3)]
		cat := []string{"society", "sports", "science", "technology", "entertainment", "economy", "other", "not_news"}[rand.Intn(8)]
		t1 := time.Unix(rand.Int63(), 0)
		t2 := time.Unix(rand.Int63(), 0)
		neigh := int(rand.Int31())
		clusterID := randomClusterID()
		vector := randomVector()

		if rand.Float32() < 0.5 {
			clusterID = zeroClusterID()
		}

		encoded, err := encodeArticle(lang, cat, t1, t2, neigh, clusterID, vector)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		lang_, cat_, t1_, t2_, neigh_, clusterID_, vector_, err := decodeArticle(encoded)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		if (lang != "ru" && lang != "en") || cat == "not_news" {
			if cat_ != "not_news" {
				t.Fatalf("Error decoding or encoding article")
			}
		} else {
			if lang != lang_ || cat != cat_ || neigh != neigh_ || clusterID != clusterID_ || t1.Unix() != t1_.Unix() || t2.Unix() != t2_.Unix() {
				t.Fatalf("Error decoding or encoding article")
			}
			for i := range vector {
				if vector[i] != vector_[i] {
					t.Fatalf("Error decoding or encoding article")		
				}
			}
		}
	}
}

func TestDeletingOldArticles(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
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

	addArticle("1.html", "1", "ru", "society", time.Now(), time.Now().Add(3 * time.Second), []float32{0})
	if deleteOldArticles() != nil {
		t.Errorf("Error deleting old articles")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte("1.html")) == nil {
			t.Errorf("Error deleting old articles")		
		}
		return nil
	}) != nil {
		t.Errorf("Error viewing database")
	}

	time.Sleep(time.Second)

	if deleteOldArticles() != nil {
		t.Errorf("Error deleting old articles")
	}
	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte("1.html")) == nil {
			t.Errorf("Error deleting old articles")		
		}
		return nil
	}) != nil {
		t.Errorf("Error viewing database")
	}

	time.Sleep(3 * time.Second)

	if deleteOldArticles() != nil {
		t.Errorf("Error deleting old articles")
	}
	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		if articles_bucket.Get([]byte("1.html")) != nil {
			t.Errorf("Error deleting old articles")		
		}
		return nil
	}) != nil {
		t.Errorf("Error viewing database")
	}


}

func TestClustering(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
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

	if ru_dbscan_eps != 0.1 || en_dbscan_eps != 0.1 || ru_dbscan_minpts != 3 || en_dbscan_minpts != 3 {
		t.Fatalf("Invalid hyperparameters:\neps: 0.1\nminpts: 3\ndistance: euclidean")
	}

	name1 := "111.html"
	name2 := "112.html"
	name3 := "113.html"
	name4 := "114.html"

	var cluster1, cluster2 string

	fmt.Println("add 1: {0, 0}")

	if addArticle(name1, "1", "ru", "society", time.Now(), time.Now(), []float32{0.0, 0.0}) != 201 {
		t.Fatalf("Error adding article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		lang, category, t1, t2, neigh, clusterID, vector, err := decodeArticle(articles_bucket.Get([]byte(name1)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 0 || clusterID != zeroClusterID() {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(zeroClusterID())) != nil {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name1))) != "1" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 1: " + clusterID)

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	fmt.Println("add 2: {0, 0.05}")

	if addArticle(name2, "2", "ru", "society", time.Now(), time.Now(), []float32{0.0, 0.05}) != 201 {
		t.Fatalf("Error adding article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		// check 1
		lang, category, t1, t2, neigh, clusterID, vector, err := decodeArticle(articles_bucket.Get([]byte(name1)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		fmt.Println(lang, category, t1, t2, neigh, clusterID, vector)
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 1 || clusterID != zeroClusterID() {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(zeroClusterID())) != nil {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name1))) != "1" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 1: " + clusterID)

		// check 2
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name2)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0.05 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 1 || clusterID != zeroClusterID() {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(name2)) != nil {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(zeroClusterID())) != nil {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name2))) != "2" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 2: " + clusterID)

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	fmt.Println("add 3: {0.05, 0}")

	if addArticle(name3, "3", "ru", "society", time.Now(), time.Now(), []float32{0.05, 0}) != 201 {
		t.Fatalf("Error adding article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		// check 1
		lang, category, t1, t2, neigh, clusterID, vector, err := decodeArticle(articles_bucket.Get([]byte(name1)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}

		cluster1 = clusterID
		if clusterID == zeroClusterID() {
			t.Fatalf("Clustering error")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 2 || clusterID != cluster1 {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(zeroClusterID())) != nil {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name1))) != "1" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 1: " + clusterID)

		// check 2
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name2)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0.05 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 2 || clusterID != cluster1 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name2))) != "2" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 2: " + clusterID)

		// check 3
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name3)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0.05 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 2 || clusterID != cluster1 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name3))) != "3" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 3: " + clusterID)

		// check cluster1
		if clusters_bucket.Get([]byte(cluster1)) == nil {
			t.Fatalf("Clustering error")
		}

		var articles []string
		err = json.Unmarshal(clusters_bucket.Get([]byte(cluster1)), &articles)
		if err != nil {
			t.Fatalf("Error parsing json")
		}
		if len(articles) != 3 {
			t.Fatalf("Clustering error")
		}
		fmt.Printf("Cluster1: %v, %v, %v\n", articles[0], articles[1], articles[2])

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	fmt.Println("add 4: {0.05, 0.05}")

	if addArticle(name4, "4", "ru", "society", time.Now(), time.Now(), []float32{0.05, 0.05}) != 201 {
		t.Fatalf("Error adding article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		// check 1
		lang, category, t1, t2, neigh, clusterID, vector, err := decodeArticle(articles_bucket.Get([]byte(name1)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		cluster2 = clusterID

		if clusterID == zeroClusterID() {
			t.Fatalf("Clustering error")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 3 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if clusters_bucket.Get([]byte(zeroClusterID())) != nil {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name1))) != "1" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 1: " + clusterID)

		// check 2
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name2)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0.05 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 3 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name2))) != "2" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 2: " + clusterID)

		// check 3
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name3)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0.05 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 3 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name3))) != "3" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 3: " + clusterID)

		// check 4
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name4)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0.05 || vector[1] != 0.05 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 3 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name4))) != "4" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 4: " + clusterID)

		// check cluster1
		if clusters_bucket.Get([]byte(cluster1)) != nil {
			t.Fatalf("Clustering error")
		}

		// check cluster2
		if clusters_bucket.Get([]byte(cluster2)) == nil {
			t.Fatalf("Clustering error")
		}

		var articles []string
		err = json.Unmarshal(clusters_bucket.Get([]byte(cluster2)), &articles)
		if err != nil {
			t.Fatalf("Error parsing json")
		}
		if len(articles) != 4 {
			t.Fatalf("Clustering error")
		}
		fmt.Printf("Cluster2: %v, %v, %v, %v\n", articles[0], articles[1], articles[2], articles[3])

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	fmt.Println("update 4: {0, 0.125}")

	if addArticle(name4, "4", "ru", "society", time.Now(), time.Now(), []float32{0, 0.125}) != 204 {
		t.Fatalf("Error adding article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))

		// check 1
		_, _, _, _, neigh, clusterID, _, err := decodeArticle(articles_bucket.Get([]byte(name1)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if neigh != 2 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}
		fmt.Println("clusterID 1: " + clusterID)

		// check 2
		_, _, _, _, neigh, clusterID, _, err = decodeArticle(articles_bucket.Get([]byte(name2)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if neigh != 3 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}
		fmt.Println("clusterID 2: " + clusterID)

		// check 3
		_, _, _, _, neigh, clusterID, _, err = decodeArticle(articles_bucket.Get([]byte(name3)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if neigh != 2 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}
		fmt.Println("clusterID 3: " + clusterID)

		// check 4
		_, _, _, _, neigh, clusterID, _, err = decodeArticle(articles_bucket.Get([]byte(name4)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if neigh != 1 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}
		fmt.Println("clusterID 4: " + clusterID)

		

		// check cluster1
		if clusters_bucket.Get([]byte(cluster1)) != nil {
			t.Fatalf("Clustering error")
		}

		// check cluster2
		if clusters_bucket.Get([]byte(cluster2)) == nil {
			t.Fatalf("Clustering error")
		}

		var articles []string
		err = json.Unmarshal(clusters_bucket.Get([]byte(cluster2)), &articles)
		if err != nil {
			t.Fatalf("Error parsing json")
		}
		if len(articles) != 4 {
			t.Fatalf("Clustering error")
		}
		fmt.Printf("Cluster2: %v, %v, %v, %v\n", articles[0], articles[1], articles[2], articles[3])

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	
	fmt.Println("delete 1")

	if deleteArticle(name1) != 204 {
		t.Fatalf("Error deleting article")
	}
	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		// check 1
		if articles_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Error deleting article")	
		}

		if titles_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Error deleting article")	
		}

		// check 2
		lang, category, t1, t2, neigh, clusterID, vector, err := decodeArticle(articles_bucket.Get([]byte(name2)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		fmt.Println(clusterID, neigh)
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0.05 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 2 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name2))) != "2" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 2: " + clusterID)

		// check 3
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name3)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0.05 || vector[1] != 0 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 1 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name3))) != "3" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 3: " + clusterID)

		// check 4
		lang, category, t1, t2, neigh, clusterID, vector, err = decodeArticle(articles_bucket.Get([]byte(name4)))
		if err != nil {
			t.Fatalf("Error decoding article")
		}
		if lang != "ru" || category != "society" || vector[0] != 0 || vector[1] != 0.125 {
			t.Fatalf("Encoding or decoding error")
		} 
		if t1.Unix() != time.Now().Unix() || t2.Unix() != time.Now().Unix() {
			t.Fatalf("Time error")
		}
		if neigh != 1 || clusterID != cluster2 {
			t.Fatalf("Clustering error")
		}

		if string(titles_bucket.Get([]byte(name4))) != "4" {
			t.Fatalf("Error adding article")
		}

		fmt.Println("clusterID 4: " + clusterID)

		// check cluster1
		if clusters_bucket.Get([]byte(cluster1)) != nil {
			t.Fatalf("Clustering error")
		}

		// check cluster2
		if clusters_bucket.Get([]byte(cluster2)) == nil {
			t.Fatalf("Clustering error")
		}

		var articles []string
		err = json.Unmarshal(clusters_bucket.Get([]byte(cluster2)), &articles)
		if err != nil {
			t.Fatalf("Error parsing json")
		}
		if len(articles) != 3 {
			t.Fatalf("Clustering error")
		}
		fmt.Printf("Cluster2: %v, %v, %v\n", articles[0], articles[1], articles[2])

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

	fmt.Println("delete 2")
	if deleteArticle(name2) != 204 {
		t.Fatalf("Error deleting article")
	}

	fmt.Println("delete 3")
	if deleteArticle(name3) != 204 {
		t.Fatalf("Error deleting article")
	}

	fmt.Println("delete 4")
	if deleteArticle(name4) != 204 {
		t.Fatalf("Error deleting article")
	}

	fmt.Println("delete 4 again")
	if deleteArticle(name4) != 404 {
		t.Fatalf("Error deleting article")
	}

	if db.View(func(tx *bolt.Tx) error {
		articles_bucket := tx.Bucket([]byte("articles"))
		clusters_bucket := tx.Bucket([]byte("ru_society"))
		titles_bucket := tx.Bucket([]byte("titles"))

		// check 1
		if articles_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Error deleting article")	
		}

		if titles_bucket.Get([]byte(name1)) != nil {
			t.Fatalf("Error deleting article")	
		}

		// check 2
		if articles_bucket.Get([]byte(name2)) != nil {
			t.Fatalf("Error deleting article")	
		}

		if titles_bucket.Get([]byte(name2)) != nil {
			t.Fatalf("Error deleting article")	
		}

		// check 3
		if articles_bucket.Get([]byte(name3)) != nil {
			t.Fatalf("Error deleting article")	
		}

		if titles_bucket.Get([]byte(name3)) != nil {
			t.Fatalf("Error deleting article")	
		}
		
		// check 4
		if articles_bucket.Get([]byte(name4)) != nil {
			t.Fatalf("Error deleting article")	
		}

		if titles_bucket.Get([]byte(name4)) != nil {
			t.Fatalf("Error deleting article")	
		}
	

		// check cluster1
		if clusters_bucket.Get([]byte(cluster1)) != nil {
			t.Fatalf("Clustering error")
		}

		// check cluster2
		if clusters_bucket.Get([]byte(cluster2)) != nil {
			t.Fatalf("Clustering error")
		}

		return nil
	}) != nil {
		t.Fatalf("Error viewing database")	
	}

}