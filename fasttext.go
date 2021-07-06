package main

// #cgo LDFLAGS: -L. -lfasttext -lstdc++ -lm
// #include <stdlib.h>
// void load_model(char *name, char *path);
// int predict(char* name, char *query, float *prob, char **buf, int *count, int k, int buf_sz);
// void sentence_vector(char *name, char *query, float *buf);
import "C"
import (
	"unsafe"
	"errors"
)

func LoadModel(name, path string) {
	p1 := C.CString(name)
	p2 := C.CString(path)

	C.load_model(p1, p2)

	C.free(unsafe.Pointer(p1))
	C.free(unsafe.Pointer(p2))
}

func Predict(name, sentence string, topN int) (map[string]float32, error) {
	result := make(map[string]float32)
	// need for fasttext
	sentence += "\n"

	cprob := make([]C.float, topN, topN)
	buf := make([]*C.char, topN, topN)
	var resultCnt C.int
	for i := 0; i < topN; i++ {
		buf[i] = (*C.char)(C.calloc(64, 1))
	}

	np := C.CString(name)
	data := C.CString(sentence)

	ret := C.predict(np, data, &cprob[0], &buf[0], &resultCnt, C.int(topN), 64)
	if ret != 0 {
		return result, errors.New("error in prediction")
	} else {
		for i := 0; i < int(resultCnt); i++ {
			result[C.GoString(buf[i])] = float32(cprob[i])
		}
	}

	C.free(unsafe.Pointer(data))
	C.free(unsafe.Pointer(np))
	for i := 0; i < topN; i++ {
		C.free(unsafe.Pointer(buf[i]))
	}

	return result, nil
}

func SentenceVector(name, sentence string) []float32 {
	result := make([]float32, dim)
	// need for fasttext
	sentence += "\n"
	
	np := C.CString(name)
	data := C.CString(sentence)
	buf := make([]C.float, dim, dim)

	C.sentence_vector(np, data, &buf[0])

	C.free(unsafe.Pointer(data))
	C.free(unsafe.Pointer(np))

	for i := 0;i<dim;i++ {
		result[i] = float32(buf[i])
	}

	return result
}