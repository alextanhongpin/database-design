# Search Engine

This is a basic markdown search engine for this directory using sqlite3 fts5.



```bash
$ make compile
$ make init
$ make index
$ make search q=uuid
$ make search q='email*' # Extra quote is required when using the '*'!
$ make search q='email+varchar' # No space between the '+' icon!
```
