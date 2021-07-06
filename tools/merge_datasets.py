import json
import random
import tarfile
import numpy as np
import pandas as pd
from preprocess import preprocess

data_en = {}
data_ru = {}

train_ratio = 0.8
fasttext = True
fasttext_news = True
categories = ['society', 'sports', 'science', 'technology', 'entertainment', 'economy', 'other', 'not_news']

for cat in categories:
	data_en[cat] = []
	data_ru[cat] = []


def load_en_tg_train():
	tar = tarfile.open("datasets/en_train.tar.gz")
	data = None
	for member in tar.getmembers():
		f = tar.extractfile(member)
		if f is not None:
			data = json.load(f)
	for item in data:
		lang = item["language"]
		if lang != "en":
			continue
		text = item["title"] + item["description"] + item["text"]
		data_en[item["category"]].append(text)


def load_en_tg_test():
	tar = tarfile.open("datasets/en_test.tar.gz")
	data = None
	for member in tar.getmembers():
		f = tar.extractfile(member)
		if f is not None:
			data = json.load(f)
	for item in data:
		lang = item["language"]
		if lang != "en":
			continue
		text = item["title"] + item["description"] + item["text"]
		data_en[item["category"]].append(text)


def load_ru_tg_train():
	tar = tarfile.open("datasets/ru_train.tar.gz")
	data = None
	for member in tar.getmembers():
		f = tar.extractfile(member)
		if f is not None:
			data = json.load(f)
	for item in data:
		lang = item["language"]
		if lang != "ru":
			continue
		text = item["title"] + item["description"] + item["text"]
		data_ru[item["category"]].append(text)


def load_ru_tg_test():
	tar = tarfile.open("datasets/ru_test.tar.gz")
	data = None
	for member in tar.getmembers():
		f = tar.extractfile(member)
		if f is not None:
			data = json.load(f)
	for item in data:
		lang = item["language"]
		if lang != "ru":
			continue
		text = item["title"] + " " + item["description"] + " " + item["text"]
		data_ru[item["category"]].append(text)


def load_en_cat_train():
	df = pd.read_csv("datasets/en_cat_train.tsv", sep="\t")
	for item in df.values:
		title = item[2]
		text = item[1]
		for category in set(item[5:8]):
			if category in categories:
				data_en[category].append(title)


def load_en_cat_test():
	df = pd.read_csv("datasets/en_cat_test.tsv", sep="\t")
	for item in df.values:
		title = item[2]
		text = item[1]
		for category in set(item[5:8]):
			if category in categories:
				data_en[category].append(title)


def load_ru_cat_train():
	df = pd.read_csv("datasets/ru_cat_train.tsv", sep="\t")
	for item in df.values:
		title = item[2]
		text = item[1]
		for category in set(item[3:6]):
			if category in categories:
				data_ru[category].append(title)


def load_ru_cat_test():
	df = pd.read_csv("datasets/ru_cat_test.tsv", sep="\t")
	for item in df.values:
		title = item[2]
		text = item[1]
		for category in set(item[3:6]):
			if category in categories:
				data_ru[category].append(title)


def load_entry_en(entry):
	df = pd.read_csv("datasets/%s_en_td.tsv" % entry, sep="\t")
	for item in df.values:
		data_en[item[1]].append(item[0])


def load_entry_ru(entry):
	df = pd.read_csv("datasets/%s_ru_td.tsv" % entry, sep="\t")
	for item in df.values:
		data_ru[item[1]].append(item[0])

load_en_cat_train()
load_en_cat_test()
load_ru_cat_train()
load_ru_cat_test()

# load_en_tg_train()
# load_en_tg_test()
# load_ru_tg_train()
# load_ru_tg_test()

# load_entry_en("entry1169")
# load_entry_ru("entry1169")
load_entry_en("entry1171")
load_entry_ru("entry1171")

len_en = 0
print("data_en:")
for cat in categories:
	len_en += len(data_en[cat])
	print(cat, len(data_en[cat]))

len_ru = 0
print("data_ru:")
for cat in categories:
	len_ru += len(data_ru[cat])
	print(cat, len(data_ru[cat]))

# out = np.zeros(shape=(len_ru + len_en, 2), dtype=object)
# cnt = 0
# for cat in categories:
# 	for item in data_en[cat]:
# 		out[cnt][0] = item
# 		out[cnt][1] = categories.index(cat)
# 		cnt += 1
# for cat in categories:
# 	for item in data_ru[cat]:
# 		out[cnt][0] = item
# 		out[cnt][1] = categories.index(cat)
# 		cnt += 1

# np.random.shuffle(out)

# th = int(train_ratio * len(out))

# pd.DataFrame(out[:th]).to_csv("datasets/train.tsv", index=False, sep="\t")
# pd.DataFrame(out[th:]).to_csv("datasets/test.tsv", index=False, sep="\t")


# out_en = []
# for cat in categories:
# 	for item in data_en[cat]:
# 		out_en.append((item, categories.index(cat)))

# out_ru = []
# for cat in categories:
# 	for item in data_ru[cat]:
# 		out_ru.append((item, categories.index(cat)))

# random.shuffle(out_en)
# random.shuffle(out_ru)

# th_en = int(train_ratio * len(out_en))
# th_ru = int(train_ratio * len(out_ru))

# pd.DataFrame(out_en[:th_en]).to_csv("datasets/train_en.tsv", index=False, sep="\t")
# pd.DataFrame(out_en[th_en:]).to_csv("datasets/test_en.tsv", index=False, sep="\t")

# pd.DataFrame(out_ru[:th_ru]).to_csv("datasets/train_ru.tsv", index=False, sep="\t")
# pd.DataFrame(out_ru[th_ru:]).to_csv("datasets/test_ru.tsv", index=False, sep="\t")

if fasttext:
	out_en = []
	for cat in categories:
		for item in data_en[cat]:
			# if cat == "not_news": continue
			out_en.append("__label__%s %s" % (cat, preprocess(item)))

	out_ru = []
	for cat in categories:
		for item in data_ru[cat]:
			# if cat == "not_news": continue
			out_ru.append("__label__%s %s" % (cat, preprocess(item)))

	random.shuffle(out_en)
	random.shuffle(out_ru)

	th_en = int(train_ratio * len(out_en))
	th_ru = int(train_ratio * len(out_ru))

	with open('datasets/text_en.train', 'w') as f:
		f.write('\n'.join(out_en[:th_en]))

	with open('datasets/text_en.valid', 'w') as f:
		f.write('\n'.join(out_en[th_en:]))

	with open('datasets/text_ru.train', 'w') as f:
		f.write('\n'.join(out_ru[:th_ru]))

	with open('datasets/text_ru.valid', 'w') as f:
		f.write('\n'.join(out_ru[th_ru:]))

# if fasttext_news:
# 	def news(cat):
# 		if cat == "not_news": return "not_news"
# 		return "news"

# 	out_en = []
# 	for cat in categories:
# 		for item in data_en[cat]:
# 			out_en.append("__label__%s %s" % (news(cat), preprocess(item)))

# 	out_ru = []
# 	for cat in categories:
# 		for item in data_ru[cat]:
# 			out_ru.append("__label__%s %s" % (news(cat), preprocess(item)))

# 	random.shuffle(out_en)
# 	random.shuffle(out_ru)

# 	th_en = int(train_ratio * len(out_en))
# 	th_ru = int(train_ratio * len(out_ru))

# 	with open('datasets/text_en_td.train', 'w') as f:
# 		f.write('\n'.join(out_en[:th_en]))

# 	with open('datasets/text_en_td.valid', 'w') as f:
# 		f.write('\n'.join(out_en[th_en:]))

# 	with open('datasets/text_ru_td.train', 'w') as f:
# 		f.write('\n'.join(out_ru[:th_ru]))

# 	with open('datasets/text_ru_td.valid', 'w') as f:
# 		f.write('\n'.join(out_ru[th_ru:]))