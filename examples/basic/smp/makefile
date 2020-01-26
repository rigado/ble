native: clean
	go build -o smp-example

arm: clean
	env GOOS=linux GOARCH=arm go build -o smp-example-arm

clean:
	rm -f smp-example smp-example-arm
