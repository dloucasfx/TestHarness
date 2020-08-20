# SWAT-18090

Example to build and extract the binary from the image
```
docker build -t dlouca/testharness:latest .
id=$(docker create image-name)
docker cp $id:/go/src/app/hostObserverTestHarness .
docker rm -v $id
```

