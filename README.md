# file-transfer

To create all servers and clients

```bash
make all
```

This will create 
```text
\bin
...\jonathan
......client
......server
...\wilson
......client
......server
```

To run server / client
```bash
./bin/wilson/client localhost:9898 put /Users/jonathansamuel/projects/cs-677/l3-wilson/clientStuff/test.txt
./bin/jonathan/server 9898 ./stuff 
```
