Add `schemaspy.properties`:


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

```make
schemaspy:
	docker run -v "$(shell pwd)/schemaspy:/output" -v "$(shell pwd)/schemaspy.properties:/schemaspy.properties" schemaspy/schemaspy:snapshot
```
