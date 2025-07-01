# Embedding

We want to store embeddings for media (text/audio/image etc) in the database.

We can use `pgvector` for that.

However, this article is not about the usage, but how to allow our db (or any vector database) to handle changes in embedding.


Why would the embedding change?

- the document uploaded is updated
- we are using a new model to generate embedding

For the former, the solution is simple. Just delete the old embedding and create new one.

Unfortunately upsert doesn't work well, because if the document is shorter, we will generate less parts.

If there are concern that the initial deletion might interrupt search experience, we can first insert the new embeddings, but then add a metadata such as updated at. Then we can delete the older version.


For the latter, it is probably best to just create a new table and backfill the results, e.g. by iterating all the older records and updating them.

