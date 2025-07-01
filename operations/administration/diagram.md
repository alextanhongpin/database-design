# View Schema relationship with sequel pro.

```
# You need to have sequel pro installed.
$ brew install graphviz
```

```
Sequel Pro > File > Export ... > .dot
# This should export a file called database_name.dot
$ dot -Tpng database_name.dot > database_name.png
```
