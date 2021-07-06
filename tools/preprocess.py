pattern = '-’0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZабвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ'

def preprocess(in_text):
	text = ""
	for char in in_text:
		if char in pattern:
			text += char
		elif char in ",.!?":
			text += " " + char + " "
		else:
			text += " "
			
	text = text.lower()
	return " ".join([i for i in text.split(' ') if i != ''])