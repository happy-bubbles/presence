#setup
rm -rf out
mkdir out
cp -rf static_html out/

cp VERSION out/static_html/

build_os_arch ()
{
	OS="$1"
	ARCH="$2"
	rm out/presence
	env GOOS=$OS GOARCH=$ARCH go build -o out/presence
	cd out
	zip -r presence_${OS}_${ARCH}.zip * presence
	mv presence_${OS}_${ARCH}.zip ..
	cd ..
	echo "finished building presence_${OS}_${ARCH}.zip"
}

build_os_arch "darwin" "amd64"
build_os_arch "linux" "386"
build_os_arch "linux" "amd64"
build_os_arch "linux" "arm"
build_os_arch "linux" "arm64"
build_os_arch "windows" "386"
build_os_arch "windows" "amd64"

