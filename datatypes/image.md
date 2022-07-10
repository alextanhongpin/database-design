# Image URLs

There are no such thing as image data type. Using plain `text` is sufficient for most cases, however, before we conclude that `text` is the best solution, let's think of other alternative to storing image in database.

## Storing BLOBs

Yes, it is possible to store blobs in Postgres. Google on why this is not recommended.

## Base64 string

Unfortunately, before I know about File System Storage like AWS S3 or Minio, this is what I do to simplify development. Images are converted to base64 strings and stored in the database.



## One or Many Images?

WIP

- storing image in a column vs separate tables?

- do you have infinite number of images? 

- for a project I worked on, we just need three image sizes. Storing them as columns therefore makes more sense than a separate table - we manage to avoid joining the tables.

- don't store unconstructed image. Just store the full image url. An image stored in db should just be a path to the location of the image, nothing more.
