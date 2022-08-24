while true; do
    curl localhost:9090/InsertOne
    curl localhost:9090/DeleteOne
    curl localhost:9090/InsertMany
    curl localhost:9090/DeleteMany
    curl localhost:9090/Find
    curl localhost:9090/UpdateAndReplace
    sleep 5
done
