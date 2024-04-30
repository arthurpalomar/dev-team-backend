# Actocracy Backend Server

## How to tun it

### Create Docker Network (just once)

```bash
docker network create actocracy
```

### Run actocracy-backend server in the current terminal session

```bash
docker-compose up --build --scale proxy=0
```

### But better run it as a service, in detached mode:

```bash
docker-compose up --build --scale proxy=0
```

### Redis Commander is listening to:

[http://localhost:8081](http://localhost:8081)

### Launch php dev server to access the socket.io test page at:

[http://localhost:8080](http://localhost:8080)

```bash
cd %project folder%
php -S localhost:8080
```

Might require php installation, for MacOS it is: ```brew install php```

## How to build actocracy-backend from scratch

#### For MacOS set architecture to `darwin` in the `Taskfile.yml`

```bash
#mkdir -p bin log
go mod download
go get github.com/go-task/task
task build
```

### To run the server:

```bash
./bin/actocracy
```
