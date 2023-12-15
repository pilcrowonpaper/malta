rm -rf bin
cd src

echo 'building darwin-amd64...'
GOOS=darwin GOARCH=amd64 go build -o ../bin/darwin-amd64/malta
echo 'building darwin-arm64...'
GOOS=darwin GOARCH=arm64 go build -o ../bin/darwin-arm64/malta

echo 'building linux-amd64...'
GOOS=linux GOARCH=amd64 go build -o ../bin/linux-amd64/malta
echo 'building linux-arm64...'
GOOS=linux GOARCH=arm64 go build -o ../bin/linux-arm64/malta

echo 'building windows-amd64...'
GOOS=windows GOARCH=amd64 go build -o ../bin/windows-amd64/malta
echo 'building windows-386...'
GOOS=windows GOARCH=386 go build -o ../bin/windows-386/malta

cd ..
cd bin
for dir in $(ls -d *); do
    tar cfzv "$dir".tgz $dir
    rm -rf $dir
done
cd ..

echo 'done!'
