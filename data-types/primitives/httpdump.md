There are quite a number of users that uses database as a workflow engine, particularly when involving API calls 

They store the status of the API, as well as the request and/or response of the API in the table.


An alternative format is to store the http dump, which jncludes the request/response line, headers and body.

This has several advantages
- allows us to replay the request
- battle proof against API versioning or changes
- human readable
- adds more context, as compared to just the request/response body


Some disadvantages includes
- accidentally committing secrets/api key in headers

if there are api keys or tokens, masked them or substitute them with a clear message that they are needed to perform the api call, dont just delete the headers.


For example, you can use the message `**REDACTED**` or `**MASKED**`.


## Alternative workflow

Woth postgres listen notify, you can simplify the workflow. Typical workflow is this

1. build a request on application side
2. save to db
3. make actual request
4. update response to db


with listen notify, you can decouple it

1. build a request
2. save to db
3. a worker will get the notification and handle api call
4. update response to db

one advantage is you can retry the api calls just by inserting a new data or updating the row.
