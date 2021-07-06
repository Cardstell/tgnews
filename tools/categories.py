import pandas as pd
import numpy as np
import hashlib, os, shutil
from preprocess import preprocess
from keras.layers.wrappers import Bidirectional
from keras.models import Sequential, Model
from keras.layers import Dense, Embedding, LSTM, Input, SpatialDropout1D, BatchNormalization, Dropout
from keras.layers.normalization import BatchNormalization
from keras.optimizers import Adam
import tensorflow as tf
from keras import regularizers
import keras.backend as K
from attention import AttentionWithContext

lang = "en"
version = "v2"
n_categories = 8

learning_rate = 0.0015
decay = 5e-6
epochs = 50
batch_size = 128

sess = tf.Session()
K.set_session(sess)

with open('tmp/train_hash_%s' % lang, 'r') as f:
	last_train_hashsum = f.read()

with open('tmp/test_hash_%s' % lang, 'r') as f:
	last_test_hashsum = f.read()

train_hashsum = hashlib.md5(open("datasets/train_%s.tsv" % lang, 'rb').read()).hexdigest()
test_hashsum = hashlib.md5(open("datasets/test_%s.tsv" % lang, 'rb').read()).hexdigest()

if train_hashsum != last_train_hashsum:
	train_data = pd.read_csv("datasets/train_%s.tsv" % lang, sep="\t").values
	train_x = train_data[:,0]
	train_y = train_data[:,1]

	for i in range(len(train_x)):
		train_x[i] = preprocess(train_x[i])

	with open('tmp/train_x_%s' % lang, 'w') as f:
		f.write("\n".join(train_x))

	np.array(train_y, dtype=np.int32).tofile('tmp/train_y_%s' % lang)
	
	with open('tmp/train_hash_%s' % lang, 'w') as f:
		f.write(train_hashsum)
else:
	print('Used cached file: tmp/train_x_%s' % lang)
	with open('tmp/train_x_%s' % lang, 'r') as f:
		train_x = f.read().splitlines()
	print('Used cached file: tmp/train_y_%s' % lang)
	train_y = np.fromfile('tmp/train_y_%s' % lang, dtype=np.int32)

if test_hashsum != last_test_hashsum:
	test_data = pd.read_csv("datasets/test_%s.tsv" % lang, sep="\t").values
	test_x = test_data[:,0]
	test_y = test_data[:,1]

	for i in range(len(test_x)):
		test_x[i] = preprocess(test_x[i])

	with open('tmp/test_x_%s' % lang, 'w') as f:
		f.write("\n".join(test_x))

	np.array(test_y, dtype=np.int32).tofile('tmp/test_y_%s' % lang)
	
	with open('tmp/test_hash_%s' % lang, 'w') as f:
		f.write(test_hashsum)
else:
	print('Used cached file: tmp/test_x_%s' % lang)
	with open('tmp/test_x_%s' % lang, 'r') as f:
		test_x = f.read().splitlines()
	print('Used cached file: tmp/test_y_%s' % lang)
	test_y = np.fromfile('tmp/test_y_%s' % lang, dtype=np.int32)


word_index = dict()

dim = 256
vocab_size = 0
max_len = 256

def f1(y_true, y_pred):
    def recall(y_true, y_pred):
        true_positives = K.sum(K.round(K.clip(y_true * y_pred, 0, 1)))
        possible_positives = K.sum(K.round(K.clip(y_true, 0, 1)))
        recall = true_positives / (possible_positives + K.epsilon())
        return recall

    def precision(y_true, y_pred):
        true_positives = K.sum(K.round(K.clip(y_true * y_pred, 0, 1)))
        predicted_positives = K.sum(K.round(K.clip(y_pred, 0, 1)))
        precision = true_positives / (predicted_positives + K.epsilon())
        return precision
    precision = precision(y_true, y_pred)
    recall = recall(y_true, y_pred)
    return 2*((precision*recall)/(precision+recall+K.epsilon()))

def load_embeddings(path):
	global vocab_size, embedding_matrix
	with open(path, 'r') as f:
		cnt = 0
		data = f.read().splitlines()[1:]
		
		# zero token
		vocab_size = len(data) + 1 
		embedding_matrix = np.zeros((vocab_size, dim), dtype=np.float32)
		for line in data:
			line = line.split()

			cnt += 1
			word = line[0]
			word_index[word] = cnt
			embedding_matrix[cnt] = np.array([float(i) for i in line[1:]])

def tokenize(text):
	result = np.zeros(max_len, dtype=np.int32)
	cnt = 0
	for word in text.split():
		if word in word_index:
			result[cnt] = word_index[word]
			cnt += 1
			if cnt == max_len:
				 break
	return result

def build_model():
	model = Sequential()
	model.add(Embedding(vocab_size, dim, input_length=max_len, 
		weights=[embedding_matrix], trainable=False))
	model.add(LSTM(256, input_shape=(max_len, dim)))
	# model.add(LSTM(256, input_shape=(max_len, dim)))
	model.add(Dense(256, activation='tanh'))
	model.add(Dense(n_categories, activation='softmax'))
	model.compile(loss='categorical_crossentropy', optimizer=Adam(lr=learning_rate, decay=decay),
		metrics=['accuracy', f1])
	model.summary()
	return model

def build_model2():
	input_layer = Input(shape=(max_len,), dtype='int32')
	embedding_layer = Embedding(vocab_size, dim, input_length=max_len, 
		weights=[embedding_matrix], trainable=False)(input_layer)
	# drop1 = SpatialDropout1D(0.3)(embedding_layer)
	lstm_1 = Bidirectional(LSTM(256, name='blstm_1',
		activation='tanh',
		recurrent_activation='hard_sigmoid',
		recurrent_dropout=0.0,
		dropout=0.5, 
		kernel_initializer='glorot_uniform',
		return_sequences=True),
		merge_mode='concat')(embedding_layer) # drop_1
	lstm_1 = BatchNormalization()(lstm_1)
	att_layer = AttentionWithContext()(lstm_1)
	drop3 = Dropout(0.2)(att_layer)
	thread_out = Dense(256, activation='tanh')(drop3)
	cat_out = Dense(n_categories, activation='softmax')(thread_out)
	model = Model(inputs=input_layer, outputs=cat_out)
	model.compile(loss='categorical_crossentropy', optimizer=Adam(lr=learning_rate, decay=decay),
		metrics=['accuracy', f1])
	model.summary()
	return model

def one_hot(y):
	result = np.zeros((y.shape[0], n_categories), dtype=np.float32)
	for i in range(y.shape[0]):
		result[i][y[i]] = 1
	return result

load_embeddings('models/skipgram_%s_256.vec' % lang)

train_x = np.array([tokenize(text) for text in train_x], dtype="int32")
test_x = np.array([tokenize(text) for text in test_x], dtype="int32")
train_y = one_hot(train_y)
test_y = one_hot(test_y)

model = build_model2()

model.fit(train_x, train_y, validation_data=(test_x, test_y), epochs=epochs, 
	batch_size=batch_size, shuffle=True)

model.save('models/cat_%s_%s.h5' % (lang, version))

pb_path = "models/cat_%s_%s" % (lang, version)

if os.path.exists(pb_path) and os.path.isdir(pb_path):
    shutil.rmtree(pb_path)
builder = tf.saved_model.builder.SavedModelBuilder(pb_path)
builder.add_meta_graph_and_variables(sess, ["mtag"])
builder.save()
sess.close()

print()
print("Inputs:", model.inputs)
print("Outputs:", model.outputs)