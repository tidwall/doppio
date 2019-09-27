<p align="center">
<img 
    src="logo.png" 
    width="242" height="200" border="0" alt="Doppio">
</p>

Doppio is a fast experimental LRU cache server on top of [ristretto](https://github.com/dgraph-io/ristretto), [redcon](https://github.com/tidwall/redcon), and [evio](https://github.com/tidwall/evio). With support for the Redis protocol.

## Features

- Multithreaded read and write operations.
- Simplified Redis protocol support. Most Redis clients will be able to use Doppio.
- Auto eviction of older items when the server is at optional cache capacity.
- Optional `--single-threaded` flag for single-threaded, event-loop networking mode.

## Getting Started

### Building

To start using Doppio, install Go and run `go get`:

```
$ go get -u github.com/tidwall/doppio
```

This will build the application.


### Running

Start the server by running the `doppio` application:

```
$ ./doppio

6307:M 26 Sep 17:10:50.304 * Server started on port 6380 (darwin/amd64, 12 threads, 1.0 GB capacity)
```

### Command line interface

Use the `redis-cli` application provided by the [Redis](https://github.com/antirez/redis) project.

```
$ redis-cli -p 6380
> SET hello world
OK

> GET hello
"world"

> DEL hello
(integer) 1

> GET hello
(nil)
```

### Options

Choose LRU capacity using the `-c` flag.

```sh
$ ./doppio -c 1gb      # max capactiy of 1 GB
$ ./doppio -c 16gb     # max capactiy of 16 GB
$ ./doppio -c 500mb    # max capactiy of 500 MB
```

Run in single-threaded mode using the `--single-threaded` flag.

```sh
$ ./doppio --single-threaded
```

## Performance

Using the `redis-benchmark` tool provided by the [Redis](https://github.com/antirez/redis) project we `SET` 10,000,000 random keys and then follow it up with 10,000,000 `GET` operations.

Running on a big 48 thread r5.12xlarge server at AWS.

### Doppio

```
$ redis-benchmark -p 6380 -q -t SET,GET -P 1024 -r 1000000000 -n 10000000
SET: 7886435.50 requests per second
GET: 10482180.00 requests per second
```

### Redis

```
$ redis-benchmark -p 6379 -q -t SET,GET -P 1024 -r 1000000000 -n 10000000
SET: 1171646.31 requests per second
GET: 1762114.50 requests per second
```


### Single-threaded mode

Using the `--single-threaded` flag or `GOMAXPROCS=1`.

```
$ redis-benchmark -p 6380 -q -t SET,GET -P 1024 -r 1000000000 -n 10000000
SET: 1721763.00 requests per second
GET: 1942124.62 requests per second
```

## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License
Doppio source code is available under the MIT [License](/LICENSE).
