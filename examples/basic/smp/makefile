native: clean
	go build -o explorer

arm: clean
	env GOOS=linux GOARCH=arm go build -o explorer-arm
	scp explorer-arm admin@192.168.30.230:~

clean:
	rm -f explorer explorer-arm

dep:
	dep ensure
	# donald's sketchy dep fix, I couldnt get dep to ignore things...
	# make sure you linked the root of this project to go/src/github.com/go-ble to build
	rm -rf ../../../vendor/github.com/go-ble
