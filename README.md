# Download
```sh
git clone git@github.com:lapd-golang/boilr.git 
mkdir -p $GOPATH/src/github.com/tmrts
cp -r boilr $GOPATH/src/github.com/tmrts/
cd $GOPATH/src/github.com/tmrts
go install
```

# Use it
```sh
boilr template download <github-repo-path> <template-tag>
boilr template download tmrts/boilr-license license
```