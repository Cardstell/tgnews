import requests, json, sys
import numpy as np
import pandas as pd

entry = sys.argv[1]
dates = ['20191129', '20191209', '20200131', '20200214']
categories = ['society', 'sports', 'science', 'technology', 'entertainment', 'economy', 'other', 'not_news']

data_en = {}
data_ru = {}

for cat in categories:
	data_en[cat] = []
	data_ru[cat] = []

for date in dates:
	text = {}
	for item in json.load(open('datasets/%s.json' % date, 'r')):
		# text[item['Filename']] = item['Title'] + ' ' + item['Description'] + ' ' + item['Content']
		text[item['Filename']] = item['Title'] + ' ' + item['Description']

	articles = set()

	r = requests.get('https://%s-dcround1.usercontent.dev/%s/languages/output.txt' % (entry, date))
	data = json.loads(r.text)
	for item in data:
		if item['lang_code'] == 'en':
			for i in item['articles']:
				articles.add(i)

	r = requests.get('https://%s-dcround1.usercontent.dev/%s/categories/en/output.txt' % (entry, date))
	data = json.loads(r.text)
	for item in data:
		for i in item['articles']:
			articles.remove(i)
			data_en[item['category']].append(text[i])

	for i in articles:
		data_en['not_news'].append(text[i])

	articles = set()

	r = requests.get('https://%s-dcround1.usercontent.dev/%s/languages/output.txt' % (entry, date))
	data = json.loads(r.text)
	for item in data:
		if item['lang_code'] == 'ru':
			for i in item['articles']:
				articles.add(i)

	r = requests.get('https://%s-dcround1.usercontent.dev/%s/categories/ru/output.txt' % (entry, date))
	data = json.loads(r.text)
	for item in data:
		for i in item['articles']:
			articles.remove(i)
			data_ru[item['category']].append(text[i])

	for i in articles:
		data_ru['not_news'].append(text[i])

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

out = np.zeros(shape=(len_en, 2), dtype=object)
cnt = 0
for cat in categories:
	for item in data_en[cat]:
		out[cnt][0] = item
		out[cnt][1] = cat
		cnt += 1
pd.DataFrame(out).to_csv("datasets/%s_en_td.tsv" % entry, index=False, sep="\t")

out = np.zeros(shape=(len_ru, 2), dtype=object)
cnt = 0
for cat in categories:
	for item in data_ru[cat]:
		out[cnt][0] = item
		out[cnt][1] = cat
		cnt += 1
pd.DataFrame(out).to_csv("datasets/%s_ru_td.tsv" % entry, index=False, sep="\t")
