convert: # Converts the jupyter markdown into markdown at the root directory.
	@jupyter nbconvert --output-dir=. --to markdown ./src/**.ipynb
