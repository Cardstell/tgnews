import threading, os

prefix = 'all-datasets/20200214/'
workers = 32

def send(article):
	os.system('curl -X PUT -H "Cache-Control: max-age=100000000" --upload-file "%s" localhost:10080/%s' % (
		prefix + article, article))

def worker(articles):
	for name in articles:
		send(name)

articles = os.listdir(prefix)[:20000]
threads = []
d = len(articles) // workers
for i in range(workers):
	threads.append(threading.Thread(target=worker, args=(articles[i*d:i*d+d],)))
	threads[-1].start()
for th in threads:
	th.join()