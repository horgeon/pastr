# pastr fork

**pastr** is a super-minimal URL shortener and paste tool that uses a flat-file storage and has no dependencies.

This fork differs from the main project by:
* using request headers to build URLs where needed, with support for X-Forwarded-* headers
* using individual files for storage, allowing storage of raw byte files

## Usage

- (Optional) Set `PASTR_DATA_PATH` to data storage directory.
- (Optional) Set `PASTR_KEY_LENGTH` to a number between 4 and 12 (default: 4).
- (Optional) Set `PASTR_USE_FORWARDED_HEADERS` to use X-Forwarded-Host and X-Forwarded-Proto headers when building URLs.
- Run `go run .` to start the server locally at `http://localhost:3000`.
- Now you can either use the frontend by opening the server URL in a browser, or call the API by sending a POST request to it:

```sh
curl -X POST http://localhost:3000/_new -d "Hello, World"
```

- If you used a URL as the content, then by opening the shortened link you will be redirected to the original URL; otherwise it will be displayed as plain text.
- For docker usage, check the `docker-compose.yml` file.
