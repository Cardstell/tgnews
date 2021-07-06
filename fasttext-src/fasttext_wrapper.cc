#include <iostream>
#include <istream>
#include "fasttext.h"
#include "real.h"
#include <streambuf>
#include <cstring>
#include <sstream>
#include <istream>
#include <map>

extern "C" {

struct membuf : std::streambuf
{
    membuf(char* begin, char* end) {
        this->setg(begin, begin, end);
    }
};

std::map<std::string, fasttext::FastText*> g_fasttext_model;

void load_model(char *name, char *path) {
	fasttext::FastText *model=new fasttext::FastText();
	model->loadModel(std::string(path));
	g_fasttext_model[std::string(name)]=model;
}

//get top k result
int predict(char* name, char *query, float *prob, char **buf, int *count, int k, int buf_sz) {
  membuf sbuf(query, query + strlen(query));
  std::istream in(&sbuf);

  std::vector<std::pair<fasttext::real, std::string>> predictions;
  try {

  		int32_t k = 1;
  fasttext::real threshold = 0.0;


		  g_fasttext_model.at(std::string(name))->predictLine(in, predictions, k, threshold);
		
		  int i=0;
		  for (auto it = predictions.cbegin(); it != predictions.cend() && i<k; it++) {
		    *(prob+i) = it->first;
		    strncpy(*(buf+i), it->second.c_str(), buf_sz);
			i++;
		  }
		  *count=i;
		  return 0;
  } catch (const std::exception& e) { 
		return 1;
  }
}

void sentence_vector(char *name, char *query, float *buf) {
	std::istringstream in(query);
	fasttext::Vector vec(g_fasttext_model.at(std::string(name))->args_->dim);
	g_fasttext_model.at(std::string(name))->getSentenceVector(in, vec);
	for (int i = 0;i<g_fasttext_model.at(std::string(name))->args_->dim;++i) {
		buf[i] = vec[i];
	}
}
}