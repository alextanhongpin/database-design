# Generating Documentation with Schemaspy and Schemacrawler

## Using Schemaspy

This example shows how to generate the HTML documents using __schemaspy__.

`schemaspy.properties`:
```
# type of database. Run with -dbhelp for details
schemaspy.t=pgsql
# optional path to alternative jdbc drivers.
#schemaspy.dp=path/to/drivers
# database properties: host, port number, name user, password
schemaspy.host=host.docker.internal
schemaspy.port=5432
schemaspy.db=development
schemaspy.u=john
schemaspy.p=123456
# output dir to save generated files
# schemaspy.o=path/to/output
# db scheme for which generate diagrams
schemaspy.s=public
```


`Makefile`:

```mk
.PHONY: schemaspy
schemaspy:
	docker run -v "$(shell pwd)/schemaspy:/output" -v "$(shell pwd)/schemaspy.properties:/schemaspy.properties" schemaspy/schemaspy:snapshot
```

## Using schemacrawler

`Makefile`:
```mk
schemacrawler:
	docker run \
	--mount type=bind,source="$(shell pwd)/tmp",target=/home/schcrwlr/share \
	--name schemacrawler \
	--rm -i -t \
	--entrypoint=/bin/bash \
	schemacrawler/schemacrawler:v16.9.2
	@# Paste this into the terminal.
	#schemacrawler schemacrawler connect --server postgresql --host host.docker.internal --user john --password 123456 --database development load --info-level maximum --command schema --outputformat html -o output.html
```
