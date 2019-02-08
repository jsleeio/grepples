# grepples

## what is this?

`grepples` is an utility similar to classic Unix `grep`, but intended to work
with objects stored in AWS S3 buckets. Objects are listed, retrieved and
searched in an aggressively parallelized manner to mitigate the latency hit
that may be incurred with the S3 API.

## usage

```
$ grepples -help
Usage of grepples:
  -bucket string
    	Name of S3 bucket to operate in
  -content-match string
    	String match on S3 object content
  -extra-newlines
    	Output an extra newline after each object's matches (default true)
  -fit-to-tty
    	Truncate output lines at $COLUMNS-1 characters
  -key-match string
    	String match on S3 object key
  -max-keys int
    	Maximum number of keys per page when listing S3 objects (default 1000)
  -max-workers int
    	Maximum number of processing workers (default 250)
  -object-keys
    	Include matching object keys in output (default true)
  -only-list-key-matches
    	Just print a list of objects matching -prefix and -key-match options
  -only-list-matching-objects
    	Don't print any content, just show keys of matching objects (like grep -l)
  -prefix string
    	Bucket object base prefix
  -region string
    	AWS region to operate in (default "us-west-2")
  -sort-by-key
    	Sort output by object key, lexicographically (default true)
  -tasks-ticker
    	Enable debug logging of task queue length
```

## filtering output by S3 key, with a prefix match

The `-prefix` option accepts an S3 key prefix. This is passed to the S3 API
endpoint used to list objects, ie. the filtering is performed by AWS. Useful
with S3 buckets with a well-defined key structure. For example, given the
below key structure in S3:

```
2019/01/05/00/myhourlyjob.2019-01-05T00:02:00.124Z.log
2019/01/05/01/myhourlyjob.2019-01-05T01:02:00.348Z.log
2019/01/05/02/myhourlyjob.2019-01-05T02:02:00.471Z.log
...
```

You could use a filter prefix like in the below example to pull out just
the 2019/01/05 logs:

```
$ grepples -bucket=mylogs -prefix=2018/01/05 [...other options]
```

## filtering output by S3 key, with regular expressions

The `-key-match` option accepts a [Golang regular
expression](https://github.com/google/re2/wiki/Syntax).  This regular
expression is matched against the S3 object keys. Only matching keys will be
downloaded and searched.

Following on from the `-prefix` demo; this could be used to pull out only
the logs for the `02:00` invocations of the job:

```
$ grepples -bucket=mylogs -prefix=2019/01/05 -key-match=T02:00 [...other options]
```

## todo

* modularize code a bit
* support Google Cloud Storage too
* maybe support multiple key and content matchers
* optional colourfulness when stdout is a terminal
* optional surrounding-context lines
* better error handling

## license

Copyright 2018 John Slee.  Released under the terms of the MIT license
[as included in this repository](LICENSE).
