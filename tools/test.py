import sys, json, subprocess, time, os, shutil, random
from preprocess import preprocess
from polyglot.detect import Detector

date = "20200214"

os.system("go build")

data = {}
for item in json.load(open('datasets/%s.json' % date, 'r')):
	data[item['Filename']] = item

def getLang(text):
	try:
		return Detector(text).language.code
	except:
		return "other"

if 'languages' in sys.argv:
	start = time.time()
	out = subprocess.check_output("./tgnews languages all-datasets/" + date, shell=True)
	end = time.time()
	out = out.decode('utf-8')
	out = json.loads(out)

	if os.path.exists('tmp_result'): 
		shutil.rmtree('tmp_result')
	os.mkdir('tmp_result')

	ru = []
	en = []
	other = []

	for item in out:
		os.mkdir('tmp_result/%s' % item['lang_code'])
		for article in item['articles']:
			shutil.copy("all-datasets/%s/%s" % (date, article), "tmp_result/%s/%s" % 
				(item['lang_code'], article))
			if item['lang_code'] == "ru":
				ru.append(article)
			elif item['lang_code'] == "en":
				en.append(article)
			else:
				other.append(article)

	random.shuffle(ru)
	random.shuffle(en)
	random.shuffle(other)

	N = 10

	ru = ru[:N]
	en = en[:N]
	other = other[:N]

	print('ru:')
	for i in ru:
		print(data[i]['Title'] + ' ' + data[i]['Description'])
	print()
	print('en:')
	for i in en:
		print(data[i]['Title'] + ' ' + data[i]['Description'])
	print()
	print('other:')
	for i in other:
		print(data[i]['Title'] + ' ' + data[i]['Description'])
	print()
	print("languages finished in %.3fs" % (end-start))


if 'news' in sys.argv:
	start = time.time()
	out = subprocess.check_output("./tgnews news all-datasets/" + date, shell=True)
	end = time.time()
	out = out.decode('utf-8')
	out = json.loads(out)

	not_news = set(data.keys())
	news = []

	if os.path.exists('tmp_result'): 
		shutil.rmtree('tmp_result')
	os.mkdir('tmp_result')
	os.mkdir('tmp_result/news')
	os.mkdir('tmp_result/not_news')

	for article in out['articles']:
		not_news.remove(article)
		news.append(article)
		shutil.copy("all-datasets/%s/%s" % (date, article), "tmp_result/news/%s" % article)

	for article in not_news:
		shutil.copy("all-datasets/%s/%s" % (date, article), "tmp_result/not_news/%s" % article)

	not_news = list(not_news)

	random.shuffle(news)
	random.shuffle(not_news)

	N = 1000

	news = news[:N]
	not_news = not_news[:N]

	# print('news:')
	# for i in news:
	# 	print(data[i]['Title'] + ' ' + data[i]['Description'])
	# print()
	print('not_news:')
	for i in not_news:
		text = data[i]['Title'] + ' ' + data[i]['Description'] 
		if getLang(text) not in ("ru", "en"):
			continue
		print(data[i]['Title'] + ' ' + data[i]['Description'])
	print()
	print("news finished in %.3fs" % (end-start))

if 'categories' in sys.argv:
	start = time.time()
	out = subprocess.check_output("./tgnews categories all-datasets/" + date, shell=True)
	end = time.time()
	out = out.decode('utf-8')
	out = json.loads(out)

	if os.path.exists('tmp_result'): 
		shutil.rmtree('tmp_result')
	os.mkdir('tmp_result')

	cats = {}

	for item in out:
		os.mkdir('tmp_result/%s' % item['category'])
		for article in item['articles']:
			if item['category'] not in cats:
				cats[item['category']] = []
			cats[item['category']].append(article)
			shutil.copy("all-datasets/%s/%s" % (date, article), "tmp_result/%s/%s" % 
				(item['category'], article))

	N = 20

	for cat in cats.keys():
		print(cat)
		random.shuffle(cats[cat])
		for article in cats[cat][:N]:
			print(data[article]['Title'] + ' ' + data[article]['Description'])
		print()
	
	print("categories finished in %.3fs" % (end-start))

if 'threads' in sys.argv:
	start = time.time()
	out = subprocess.check_output("./tgnews threads all-datasets/" + date, shell=True)
	end = time.time()
	out = out.decode('utf-8')
	out = json.loads(out)

	if os.path.exists('tmp_result'): 
		shutil.rmtree('tmp_result')
	os.mkdir('tmp_result')

	for thread in out:
		if len(thread['articles']) < 2:
			continue
		title = preprocess(thread['title'])
		_id = str(len(thread['articles'])) + " " + str(random.randint(0, 10000000000))
		os.mkdir('tmp_result/%s' % _id)
		for article in thread['articles']:
			shutil.copy("all-datasets/%s/%s" % (date, article), "tmp_result/%s/%s" % (_id, article))

	print("threads finished in %.3fs" % (end-start))