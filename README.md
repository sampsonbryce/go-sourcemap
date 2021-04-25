# Go Sourcemaps

A library for collecting/parsing sourcemaps and mapping javascript exceptions to their original source file

## Dependencies 

Uses https://github.com/sampsonbryce/go-vlq for decoding the Sourcemap mapping VLQ's 

## Limitations

Not all of the sourcemap v3 specification has been implemented. Most notably multi file sourcemaps are not supported.

## Usage

```bash
go-sourcemap /path/to/folder/with/sourcemaps
```

Then just make a request to localhost:8080 with an exception and itll return the exception with all the stacktrace entries mapped to the original source.

For usage in a nodejs library, checkout the client side exception catcher https://github.com/sampsonbryce/go-sourcemap-js

