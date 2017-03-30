# imagenie
Image store house

For local setup:
----------------

1. Create database imagenie in postgres

2. Set GOPATH and GOBIN environment variables

3. Update the config/settings.yml

4. Install imagnie
	go install imagenie

5. Run imagenie using:
	./bin/imagenie


Other details:
--------------

Imagenie accepts the uploaded file, then launches an async task to process and upload the image to S3.

Used the que-go library for defining and tracking the tasks. Since the status of the task is saved in database, it's persistant and not affected by server restart.

Didn't use any server push mechanism to notify clients of new upload. This could have been used, but tried to avoid complexity using websocket or eventsource.
