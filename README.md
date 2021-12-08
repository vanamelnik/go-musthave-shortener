# go-musthave-shortener
An URL shortener REST API service.

## Paths:
### GET /{id} - redirect to an initial URL
### POST / - shorten an URL provided in the body
Responses a short URL in response body.
### POST /api/shorten - shorten an URL provided in JSON object
Request body: ```{"url": "<some_url>"}```
Response: JSON object: ```{"result": "<shorten_url>"}```
### POST /api/shorten/batch - batch URL shorten
Request body: ```[{"correlation_id": "<id>", "original_url": "<URL>"}, ...]```
Response: ```[{"correlation_id": "<id>", "short_url": "<URL>"}, ...]```
### GET /user/urls - returns all URLs that have been processed in this session
Response: ```[{"short_url": "<URL>", "original_url": "<URL>"}, ...]```
### DELETE /api/user/urls - delete URLs with the keys provided
All URLs provided must be created in this session.
Request: ```["<key>", ...]```
