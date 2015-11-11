```
pushd examples/memcached
make
popd

go install ./...
${GOPATH}/bin/imagebuilder

docker load -i=output.tar

docker images | head
docker run -d -p 11211:11211 <imageid>
echo "stats" | nc 127.0.0.1 11211
```
