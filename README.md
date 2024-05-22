# Distributed Lock Using Redis (RedLocks)

## Running Multiple Redis Instances with Docker
```
docker run --name redis1 -d -p 6379:6379 redis
docker run --name redis2 -d -p 6380:6379 redis
docker run --name redis3 -d -p 6381:6379 redis
docker run --name redis4 -d -p 6382:6379 redis
docker run --name redis5 -d -p 6383:6379 redis
```

## Building the Executable

```
go build -o distributed_lock main.go
```

## Running Multiple Instances
```
./distributed_lock 1 &
./distributed_lock 2 &
./distributed_lock 3 &
./distributed_lock 4 &
```