echo 'building bin/darwin-amd64/malta'
GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64/malta
echo 'building bin/darwin-arm64/malta'
GOOS=darwin GOARCH=arm64 go build -o bin/darwin-arm64/malta

echo 'building bin/linux-amd64/malta'
GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64/malta
echo 'building bin/linux-arm64/malta'
GOOS=linux GOARCH=arm64 go build -o bin/linux-arm64/malta

echo 'building bin/windows-amd64/malta'
GOOS=windows GOARCH=amd64 go build -o bin/windows-amd64/malta
echo 'building bin/windows-386/malta'
GOOS=windows GOARCH=386 go build -o bin/windows-386/malta

cd bin
for dir in $(ls -d *); do
    tar cfzv "$dir".tgz $dir
    rm -rf $dir
done
cd ..

echo 'done!'
