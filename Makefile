#
# Copyright (c) 2016-present, Facebook, Inc.
# All rights reserved.
#
# This source code is licensed under the BSD-style license found in the
# LICENSE file in the root directory of this source tree. An additional grant
# of patent rights can be found in the PATENTS file in the same directory.
#

CXX = c++
CXXFLAGS = -pthread -std=c++11 -march=native
OBJS = args.o autotune.o matrix.o dictionary.o loss.o productquantizer.o densematrix.o quantmatrix.o vector.o model.o utils.o meter.o fasttext.o fasttext_wrapper.o
INCLUDES = -I.

opt: CXXFLAGS += -O3 -funroll-loops
opt: build

debug: CXXFLAGS += -g -O0 -fno-inline
debug: build

args.o: fasttext-src/args.cc fasttext-src/args.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/args.cc

autotune.o: fasttext-src/autotune.cc fasttext-src/autotune.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/autotune.cc

matrix.o: fasttext-src/matrix.cc fasttext-src/matrix.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/matrix.cc

dictionary.o: fasttext-src/dictionary.cc fasttext-src/dictionary.h fasttext-src/args.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/dictionary.cc

loss.o: fasttext-src/loss.cc fasttext-src/loss.h fasttext-src/matrix.h fasttext-src/real.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/loss.cc

productquantizer.o: fasttext-src/productquantizer.cc fasttext-src/productquantizer.h fasttext-src/utils.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/productquantizer.cc

densematrix.o: fasttext-src/densematrix.cc fasttext-src/densematrix.h fasttext-src/utils.h fasttext-src/matrix.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/densematrix.cc

quantmatrix.o: fasttext-src/quantmatrix.cc fasttext-src/quantmatrix.h fasttext-src/utils.h fasttext-src/matrix.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/quantmatrix.cc

vector.o: fasttext-src/vector.cc fasttext-src/vector.h fasttext-src/utils.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/vector.cc

model.o: fasttext-src/model.cc fasttext-src/model.h fasttext-src/args.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/model.cc

utils.o: fasttext-src/utils.cc fasttext-src/utils.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/utils.cc

meter.o: fasttext-src/meter.cc fasttext-src/meter.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/meter.cc

fasttext.o: fasttext-src/fasttext.cc fasttext-src/*.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/fasttext.cc

fasttext: $(OBJS) fasttext-src/fasttext.cc fasttext-src/main.cc
	$(CXX) $(CXXFLAGS) $(OBJS) fasttext-src/main.cc -o fasttext

fasttext_wrapper.o: fasttext-src/fasttext_wrapper.cc fasttext-src/fasttext.cc fasttext-src/*.h
	$(CXX) $(CXXFLAGS) -c fasttext-src/fasttext_wrapper.cc

libfasttext.a: $(OBJS)
	$(AR) rcs libfasttext.a $(OBJS)

clean:
	rm -rf *.o libfasttext.a fasttext

build: libfasttext.a
# 	go build

